package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	ashbyJobRe         = regexp.MustCompile(`jobs\.ashbyhq\.com/([^/]+)/([a-f0-9-]+)`)
	errBadAshbyURL     = errors.New("unrecognised ashby URL")
	errAshbyNotFound   = errors.New("job not found in ashby board")
	errAshbyNoNextData = errors.New("__NEXT_DATA__ not found")
	errAshbyNoJobData  = errors.New("no job data in __NEXT_DATA__")
	errAshbyHTMLStatus = errors.New("ashby page non-200 status")
)

type ashbyStatusError struct{ code int }

func (e *ashbyStatusError) Error() string { return fmt.Sprintf("ashby api status %d", e.code) }

// ashbyNextDataPosting mirrors the jobPosting shape inside Ashby's __NEXT_DATA__ JSON.
type ashbyNextDataPosting struct {
	Title                   string `json:"title"`
	DescriptionHTML         string `json:"descriptionHtml"`         //nolint:tagliatelle
	LocationName            string `json:"locationName"`            //nolint:tagliatelle
	EmploymentType          string `json:"employmentType"`          //nolint:tagliatelle
	CompensationTierSummary string `json:"compensationTierSummary"` //nolint:tagliatelle
	LocationRequirement     string `json:"locationRequirement"`     //nolint:tagliatelle
}

type ashbyNextData struct {
	Props struct {
		PageProps struct {
			JobPosting ashbyNextDataPosting `json:"jobPosting"` //nolint:tagliatelle
		} `json:"pageProps"` //nolint:tagliatelle
	} `json:"props"`
}

const ashbyBoardAPIBase = "https://api.ashbyhq.com/posting-api/job-board"

// FetchAshby parses an Ashby-hosted job posting URL via the public board API,
// falling back to __NEXT_DATA__ HTML parsing if the board is access-restricted.
func FetchAshby(ctx context.Context, rawURL string) (*ParsedJob, error) {
	match := ashbyJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadAshbyURL)
	}

	org, jobID := match[1], match[2]

	job, err := FetchAshbyFromAPI(ctx, ashbyBoardAPIBase, rawURL, org, jobID)

	var statusErr *ashbyStatusError
	if errors.As(err, &statusErr) && (statusErr.code == http.StatusUnauthorized || statusErr.code == http.StatusForbidden) {
		scraped, scrapeErr := scrapeAshbyPage(ctx, rawURL)
		if scrapeErr == nil {
			return scraped, nil
		}

		return ScrapeHTML(ctx, rawURL)
	}

	return job, err
}

type ashbyBoard struct {
	Jobs []ashbyPosting `json:"jobs"`
}

type ashbyPosting struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	LocationName    string `json:"locationName"`    //nolint:tagliatelle
	DescriptionHTML string `json:"descriptionHtml"` //nolint:tagliatelle
}

// FetchAshbyFromAPI fetches from GET {boardAPIBase}/{org} and finds the job by ID.
// boardAPIBase is injectable for tests.
func FetchAshbyFromAPI(ctx context.Context, boardAPIBase, sourceURL, org, jobID string) (*ParsedJob, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, boardAPIBase+"/"+org, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ashby api: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &ashbyStatusError{code: resp.StatusCode}
	}

	var board ashbyBoard

	decodeErr := json.NewDecoder(resp.Body).Decode(&board)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode: %w", decodeErr)
	}

	for _, posting := range board.Jobs {
		if posting.ID != jobID {
			continue
		}

		return &ParsedJob{
			Title:     posting.Title,
			Location:  posting.LocationName,
			BodyMD:    htmlToMD(posting.DescriptionHTML),
			Source:    string(ATSAshby),
			SourceURL: sourceURL,
		}, nil
	}

	return nil, fmt.Errorf("%w: %s", errAshbyNotFound, jobID)
}

// scrapeAshbyPage extracts structured job data from the __NEXT_DATA__ JSON
// embedded in Ashby's SSR'd Next.js job pages.
func scrapeAshbyPage(ctx context.Context, rawURL string) (*ParsedJob, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errAshbyHTMLStatus, resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	rawJSON := doc.Find("script#__NEXT_DATA__").Text()
	if rawJSON == "" {
		return nil, errAshbyNoNextData
	}

	var data ashbyNextData

	decodeErr := json.Unmarshal([]byte(rawJSON), &data)
	if decodeErr != nil {
		return nil, fmt.Errorf("unmarshal __NEXT_DATA__: %w", decodeErr)
	}

	posting := data.Props.PageProps.JobPosting
	if posting.Title == "" {
		return nil, errAshbyNoJobData
	}

	return &ParsedJob{
		Title:     posting.Title,
		Location:  posting.LocationName,
		SalaryRaw: posting.CompensationTierSummary,
		BodyMD:    ashbyMetaBlock(posting) + htmlToMD(posting.DescriptionHTML),
		Source:    string(ATSAshby),
		SourceURL: rawURL,
	}, nil
}

// ashbyMetaBlock builds a markdown metadata header from the structured sidebar fields.
func ashbyMetaBlock(posting ashbyNextDataPosting) string {
	type kv struct{ k, v string }

	rows := []kv{
		{"Employment Type", posting.EmploymentType},
		{"Location", posting.LocationName},
		{"Location Type", ashbyRemoteLabel(posting.LocationRequirement)},
		{"Compensation", posting.CompensationTierSummary},
	}

	var lines []string

	for _, row := range rows {
		if row.v != "" {
			lines = append(lines, "**"+row.k+":** "+row.v)
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n") + "\n\n---\n\n"
}

func ashbyRemoteLabel(req string) string {
	switch req {
	case "remote":
		return "Remote"
	case "hybrid":
		return "Hybrid"
	case "onsite":
		return "On-site"
	default:
		return ""
	}
}
