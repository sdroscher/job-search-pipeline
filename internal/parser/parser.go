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
	ATSJobgether       ATSType = "Jobgether"
	ATSJobright        ATSType = "Jobright"
	ATSLinkedIn        ATSType = "LinkedIn"
	ATSIndeed          ATSType = "Indeed"
	ATSGlassdoor       ATSType = "Glassdoor"
	ATSHTML            ATSType = "HTML"
)

// DetectATS returns the ATS type for a given URL.
func DetectATS(rawURL string) ATSType {
	lower := strings.ToLower(rawURL)

	// Order matters: more specific substrings must appear before their prefixes.
	patterns := []struct {
		substr string
		ats    ATSType
	}{
		{"boards.greenhouse.io", ATSGreenhouse},
		{"jobs.ashbyhq.com", ATSAshby},
		{"ashbyhq.com", ATSAshby},
		{"jobs.lever.co", ATSLever},
		{"bamboohr.com", ATSBambooHR},
		{"jobs.smartrecruiters.com", ATSSmartRecruiters},
		{"jobgether.com", ATSJobgether},
		{"jobright.ai", ATSJobright},
		{"linkedin.com/jobs", ATSLinkedIn},
		{"indeed.com", ATSIndeed},
		{"glassdoor.com", ATSGlassdoor},
	}

	for _, entry := range patterns {
		if strings.Contains(lower, entry.substr) {
			return entry.ats
		}
	}

	return ATSHTML
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
	case ATSJobgether:
		return FetchJobgether(ctx, rawURL)
	case ATSJobright:
		return FetchJobright(ctx, rawURL)
	case ATSLinkedIn, ATSIndeed, ATSGlassdoor:
		// These require login or are JS-heavy; fall back to HTML scrape but preserve source name
		job, err := ScrapeHTML(ctx, rawURL)
		if err != nil {
			return nil, err
		}

		job.Source = string(DetectATS(rawURL))

		return job, nil
	case ATSHTML:
		return ScrapeHTML(ctx, rawURL)
	default:
		return ScrapeHTML(ctx, rawURL)
	}
}
