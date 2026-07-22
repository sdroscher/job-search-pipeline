package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

var (
	ashbyJobRe       = regexp.MustCompile(`jobs\.ashbyhq\.com/([^/]+)/([a-f0-9-]+)`)
	errBadAshbyURL   = errors.New("unrecognised ashby URL")
	errAshbyNotFound = errors.New("job not found in ashby board")
)

type ashbyStatusError struct{ code int }

func (e *ashbyStatusError) Error() string { return fmt.Sprintf("ashby api status %d", e.code) }

const ashbyBoardAPIBase = "https://api.ashbyhq.com/posting-api/job-board"

// FetchAshby parses an Ashby-hosted job posting URL via the public board API,
// falling back to HTML scraping if the board is access-restricted (401/403).
func FetchAshby(ctx context.Context, rawURL string) (*ParsedJob, error) {
	match := ashbyJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadAshbyURL)
	}

	org, jobID := match[1], match[2]

	job, err := FetchAshbyFromAPI(ctx, ashbyBoardAPIBase, rawURL, org, jobID)

	var statusErr *ashbyStatusError
	if errors.As(err, &statusErr) && (statusErr.code == http.StatusUnauthorized || statusErr.code == http.StatusForbidden) {
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
	LocationName    string `json:"locationName"`
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

	for _, p := range board.Jobs {
		if p.ID != jobID {
			continue
		}

		return &ParsedJob{
			Title:     p.Title,
			Location:  p.LocationName,
			BodyMD:    htmlToMD(p.DescriptionHTML),
			Source:    string(ATSAshby),
			SourceURL: sourceURL,
		}, nil
	}

	return nil, fmt.Errorf("%w: %s", errAshbyNotFound, jobID)
}
