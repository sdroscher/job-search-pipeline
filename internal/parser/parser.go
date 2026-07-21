package parser

import "strings"

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
	ATSGreenhouse ATSType = "Greenhouse"
	ATSAshby      ATSType = "Ashby"
	ATSLever      ATSType = "Lever"
	ATSHTML       ATSType = "HTML"
)

// DetectATS returns the ATS type for a given URL.
func DetectATS(rawURL string) ATSType {
	lowerURL := strings.ToLower(rawURL)

	switch {
	case strings.Contains(lowerURL, "boards.greenhouse.io") || strings.Contains(lowerURL, "greenhouse.io/jobs"):
		return ATSGreenhouse
	case strings.Contains(lowerURL, "jobs.ashbyhq.com") || strings.Contains(lowerURL, "ashbyhq.com"):
		return ATSAshby
	case strings.Contains(lowerURL, "jobs.lever.co"):
		return ATSLever
	default:
		return ATSHTML
	}
}

// Parse fetches and parses a job posting URL, delegating to the appropriate ATS parser.
func Parse(rawURL string) (*ParsedJob, error) {
	switch DetectATS(rawURL) {
	case ATSGreenhouse:
		return FetchGreenhouse(rawURL)
	case ATSAshby:
		return FetchAshby(rawURL)
	case ATSLever:
		return FetchLever(rawURL)
	case ATSHTML:
		return ScrapeHTML(rawURL)
	}

	return ScrapeHTML(rawURL)
}
