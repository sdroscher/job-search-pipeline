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
	srJobRe        = regexp.MustCompile(`jobs\.smartrecruiters\.com/([^/?#]+)/([^/?#]+)`)
	errBadSRURL    = errors.New("unrecognised smartrecruiters URL")
	errSRAPIStatus = errors.New("smartrecruiters api: unexpected status")
)

const srAPIBase = "https://api.smartrecruiters.com/v1/companies"

type srCompensation struct {
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Currency string  `json:"currency"`
}

type srLocation struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Remote  bool   `json:"remote"`
}

type srEmploymentType struct {
	Label string `json:"label"`
}

type srJobSection struct {
	Text string `json:"text"`
}

type srJobSections struct {
	JobDescription        srJobSection `json:"jobDescription"` //nolint:tagliatelle
	Qualifications        srJobSection `json:"qualifications"`
	AdditionalInformation srJobSection `json:"additionalInformation"` //nolint:tagliatelle
}

type srJobAd struct {
	Sections srJobSections `json:"sections"`
}

type srResponse struct {
	Name             string           `json:"name"`
	Location         srLocation       `json:"location"`
	TypeOfEmployment srEmploymentType `json:"typeOfEmployment"` //nolint:tagliatelle
	Compensation     *srCompensation  `json:"compensation"`
	JobAd            srJobAd          `json:"jobAd"` //nolint:tagliatelle
}

// FetchSmartRecruiters parses a SmartRecruiters job URL using the public postings API.
func FetchSmartRecruiters(ctx context.Context, rawURL string) (*ParsedJob, error) {
	match := srJobRe.FindStringSubmatch(rawURL)
	if match == nil {
		return nil, fmt.Errorf("%s: %w", rawURL, errBadSRURL)
	}

	company, jobID := match[1], match[2]
	apiURL := fmt.Sprintf("%s/%s/postings/%s", srAPIBase, company, jobID)

	return FetchSmartRecruitersFromAPI(ctx, apiURL, rawURL, company, jobID)
}

// FetchSmartRecruitersFromAPI fetches from an injectable API URL (used in tests).
func FetchSmartRecruitersFromAPI(ctx context.Context, apiURL, sourceURL, _, _ string) (*ParsedJob, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sr api: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %w", resp.StatusCode, errSRAPIStatus)
	}

	var data srResponse

	decodeErr := json.NewDecoder(resp.Body).Decode(&data)
	if decodeErr != nil {
		return nil, fmt.Errorf("decode: %w", decodeErr)
	}

	return &ParsedJob{
		Title:     data.Name,
		Location:  data.Location.City,
		SalaryRaw: srSalaryRaw(data.Compensation),
		BodyMD:    srBodyMD(data),
		Source:    string(ATSSmartRecruiters),
		SourceURL: sourceURL,
	}, nil
}

func srSalaryRaw(comp *srCompensation) string {
	if comp == nil || (comp.Min == 0 && comp.Max == 0) {
		return ""
	}

	return fmt.Sprintf("%.0f-%.0f %s", comp.Min, comp.Max, comp.Currency)
}

func srBodyMD(data srResponse) string {
	sections := []string{
		data.JobAd.Sections.JobDescription.Text,
		data.JobAd.Sections.Qualifications.Text,
		data.JobAd.Sections.AdditionalInformation.Text,
	}

	var parts []string

	for _, sec := range sections {
		if strings.TrimSpace(sec) != "" {
			parts = append(parts, htmlToMD(sec))
		}
	}

	return strings.Join(parts, "\n\n")
}
