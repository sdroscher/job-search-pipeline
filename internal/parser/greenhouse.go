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
	ghJobRe = regexp.MustCompile(`boards\.greenhouse\.io/([^/]+)/jobs/(\d+)`)

	errUnrecognisedGHURL = errors.New("unrecognised greenhouse URL")
	errGHAPIStatus       = errors.New("greenhouse api: unexpected status")
)

// FetchGreenhouse parses a Greenhouse job board URL using the public boards API.
func FetchGreenhouse(ctx context.Context, rawURL string) (*ParsedJob, error) {
	match := ghJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errUnrecognisedGHURL)
	}

	board, jobID := match[1], match[2]
	apiURL := fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs/%s?questions=true", board, jobID)

	return FetchGreenhouseFromAPI(ctx, apiURL, rawURL)
}

// FetchGreenhouseFromAPI fetches a Greenhouse job from an injectable API URL (used in tests).
func FetchGreenhouseFromAPI(ctx context.Context, apiURL, sourceURL string) (*ParsedJob, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("greenhouse api: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, errGHAPIStatus)
	}

	var data struct {
		Title       string `json:"title"`
		AbsoluteURL string `json:"absolute_url"`
		Content     string `json:"content"`
		Location    struct {
			Name string `json:"name"`
		} `json:"location"`
		Departments []struct {
			Name string `json:"name"`
		} `json:"departments"`
	}

	decodeErr := json.NewDecoder(resp.Body).Decode(&data)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode: %w", decodeErr)
	}

	dept := ""
	if len(data.Departments) > 0 {
		dept = data.Departments[0].Name
	}

	return &ParsedJob{
		Title:      data.Title,
		Location:   data.Location.Name,
		Department: dept,
		BodyMD:     htmlToMD(data.Content),
		Source:     string(ATSGreenhouse),
		SourceURL:  sourceURL,
	}, nil
}
