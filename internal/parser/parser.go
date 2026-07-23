package parser

import (
	"context"
	"strings"
)

// ParsedJob holds structured data extracted from any ATS.
type ParsedJob struct {
	Title      string   `json:"title"`
	Company    string   `json:"company"`
	Location   string   `json:"location"`
	SalaryRaw  string   `json:"salary_raw"`
	Department string   `json:"department"`
	BodyMD     string   `json:"body_md"`
	Benefits   []string `json:"benefits"`
	Source     string   `json:"source"`
	SourceURL  string   `json:"source_url"`
}

// ATSType identifies which ATS platform hosts a job posting.
type ATSType string

const (
	ATSGreenhouse      ATSType = "Greenhouse"
	ATSAshby           ATSType = "Ashby"
	ATSLever           ATSType = "Lever"
	ATSBambooHR        ATSType = "BambooHR"
	ATSSmartRecruiters ATSType = "SmartRecruiters"
	ATSHTML            ATSType = "HTML"
)

// DetectATS returns the ATS type for a given URL.
func DetectATS(rawURL string) ATSType {
	lowerURL := strings.ToLower(rawURL)

	switch {
	case strings.Contains(lowerURL, "boards.greenhouse.io"):
		return ATSGreenhouse
	case strings.Contains(lowerURL, "jobs.ashbyhq.com") || strings.Contains(lowerURL, "ashbyhq.com"):
		return ATSAshby
	case strings.Contains(lowerURL, "jobs.lever.co"):
		return ATSLever
	case strings.Contains(lowerURL, "bamboohr.com"):
		return ATSBambooHR
	case strings.Contains(lowerURL, "jobs.smartrecruiters.com"):
		return ATSSmartRecruiters
	default:
		return ATSHTML
	}
}

// Parse fetches and parses a job posting URL, delegating to the appropriate ATS parser.
func Parse(ctx context.Context, rawURL string) (*ParsedJob, error) {
	switch DetectATS(rawURL) {
	case ATSGreenhouse:
		return FetchGreenhouse(ctx, rawURL)
	case ATSAshby:
		return FetchAshby(ctx, rawURL)
	case ATSLever:
		return FetchLever(ctx, rawURL)
	case ATSBambooHR:
		return FetchBambooHR(ctx, rawURL)
	case ATSSmartRecruiters:
		return FetchSmartRecruiters(ctx, rawURL)
	case ATSHTML:
		return ScrapeHTML(ctx, rawURL)
	}

	return ScrapeHTML(ctx, rawURL)
}
