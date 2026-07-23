package parser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var (
	leverJobRe        = regexp.MustCompile(`jobs\.lever\.co/([^/]+)/([a-f0-9-]+)`)
	errBadLeverURL    = errors.New("unrecognised lever URL")
	errLeverAPIStatus = errors.New("lever api non-200 status")
)

// FetchLever parses a Lever-hosted job posting URL.
func FetchLever(ctx context.Context, rawURL string) (*ParsedJob, error) {
	match := leverJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadLeverURL)
	}

	org, jobID := match[1], match[2]
	apiURL := fmt.Sprintf("https://api.lever.co/v0/postings/%s/%s?mode=json", org, jobID)

	return FetchLeverFromAPI(ctx, apiURL, rawURL)
}

// FetchLeverFromAPI fetches a Lever job from an injectable API URL (used in tests).
func FetchLeverFromAPI(ctx context.Context, apiURL, sourceURL string) (*ParsedJob, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("lever api: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", errLeverAPIStatus, resp.StatusCode)
	}

	type leverCategories struct {
		Location string `json:"location"`
		Team     string `json:"team"`
	}

	type leverList struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	}

	type leverResponse struct {
		Text        string          `json:"text"`
		Categories  leverCategories `json:"categories"`
		Description string          `json:"description"`
		Lists       []leverList     `json:"lists"`
	}

	var data leverResponse

	decodeErr := json.NewDecoder(resp.Body).Decode(&data)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode: %w", decodeErr)
	}

	bodyParts := make([]string, 0, 1+2*len(data.Lists))
	bodyParts = append(bodyParts, htmlToMD(data.Description))

	for _, list := range data.Lists {
		bodyParts = append(bodyParts, "## "+list.Text, htmlToMD(list.Content))
	}

	return &ParsedJob{
		Title:      data.Text,
		Location:   data.Categories.Location,
		Department: data.Categories.Team,
		BodyMD:     strings.Join(bodyParts, "\n\n"),
		Source:     string(ATSLever),
		SourceURL:  sourceURL,
	}, nil
}
