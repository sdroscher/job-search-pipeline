package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var (
	leverJobRe     = regexp.MustCompile(`jobs\.lever\.co/([^/]+)/([a-f0-9-]+)`)
	errBadLeverURL = errors.New("unrecognised lever URL")
)

// FetchLever parses a Lever-hosted job posting URL.
func FetchLever(rawURL string) (*ParsedJob, error) {
	match := leverJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadLeverURL)
	}

	org, jobID := match[1], match[2]
	apiURL := fmt.Sprintf("https://api.lever.co/v0/postings/%s/%s?mode=json", org, jobID)

	return FetchLeverFromAPI(apiURL, rawURL)
}

// FetchLeverFromAPI fetches a Lever job from an injectable API URL (used in tests).
func FetchLeverFromAPI(apiURL, sourceURL string) (*ParsedJob, error) {
	resp, err := http.Get(apiURL) //nolint:noctx,gosec
	if err != nil {
		return nil, fmt.Errorf("lever api: %w", err)
	}

	defer resp.Body.Close()

	var data struct {
		Text       string `json:"text"`
		Categories struct {
			Location string `json:"location"`
			Team     string `json:"team"`
		} `json:"categories"`
		Description string `json:"description"`
		Lists       []struct {
			Text    string `json:"text"`
			Content string `json:"content"`
		} `json:"lists"`
	}

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
