package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

var (
	ashbyJobRe        = regexp.MustCompile(`jobs\.ashbyhq\.com/([^/]+)/([a-f0-9-]+)`)
	errBadAshbyURL    = errors.New("unrecognised ashby URL")
	errAshbyFailed    = errors.New("ashby api returned success=false")
	errAshbyAPIStatus = errors.New("ashby api non-200 status")
)

// FetchAshby parses an Ashby-hosted job posting URL.
func FetchAshby(rawURL string) (*ParsedJob, error) {
	match := ashbyJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadAshbyURL)
	}

	org, jobID := match[1], match[2]

	return FetchAshbyFromAPI("https://api.ashbyhq.com/jobPosting.info", rawURL, org, jobID)
}

type ashbyRequest struct {
	OrgName string `json:"organizationHostedJobsPageName"` //nolint:tagliatelle
	JobID   string `json:"jobPostingId"`                   //nolint:tagliatelle
}

// FetchAshbyFromAPI fetches an Ashby job from an injectable API base URL (used in tests).
func FetchAshbyFromAPI(apiBase, sourceURL, org, jobID string) (*ParsedJob, error) {
	payload, err := json.Marshal(ashbyRequest{OrgName: org, JobID: jobID})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(apiBase, "application/json", bytes.NewReader(payload)) //nolint:noctx,gosec
	if err != nil {
		return nil, fmt.Errorf("ashby api: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errAshbyAPIStatus, resp.StatusCode)
	}

	var data struct {
		Success bool `json:"success"`
		Results struct {
			JobPosting struct {
				Title           string `json:"title"`
				DescriptionHTML string `json:"descriptionHtml"` //nolint:tagliatelle
				JobLocation     struct {
					LocationStr string `json:"locationStr"` //nolint:tagliatelle
				} `json:"jobLocation"` //nolint:tagliatelle
			} `json:"jobPosting"` //nolint:tagliatelle
		} `json:"results"`
	}

	decodeErr := json.NewDecoder(resp.Body).Decode(&data)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode: %w", decodeErr)
	}

	if !data.Success {
		return nil, errAshbyFailed
	}

	posting := data.Results.JobPosting

	return &ParsedJob{
		Title:     posting.Title,
		Location:  posting.JobLocation.LocationStr,
		BodyMD:    htmlToMD(posting.DescriptionHTML),
		Source:    string(ATSAshby),
		SourceURL: sourceURL,
	}, nil
}
