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
	ashbyJobRe        = regexp.MustCompile(`jobs\.ashbyhq\.com/([^/]+)/([a-f0-9-]+)`)
	errBadAshbyURL    = errors.New("unrecognised ashby URL")
	errAshbyNotFound  = errors.New("job not found in ashby board")
	errAshbyAPIStatus = errors.New("ashby api non-200 status")
)

const ashbyBoardAPIBase = "https://api.ashbyhq.com/posting-api/job-board"

// FetchAshby parses an Ashby-hosted job posting URL via the public board API.
func FetchAshby(ctx context.Context, rawURL string) (*ParsedJob, error) {
	match := ashbyJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadAshbyURL)
	}

	org, jobID := match[1], match[2]

	return FetchAshbyFromAPI(ctx, ashbyBoardAPIBase, rawURL, org, jobID)
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
		return nil, fmt.Errorf("%w: %d", errAshbyAPIStatus, resp.StatusCode)
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
