package parser_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
)

func (s *ParserSuite) TestDetectATS_SmartRecruiters() {
	urls := []string{
		"https://jobs.smartrecruiters.com/Grafana/123456789",
		"https://jobs.smartrecruiters.com/Acme/987654321",
	}
	for _, rawURL := range urls {
		s.Equal(parser.ATSSmartRecruiters, parser.DetectATS(rawURL), "URL: %s", rawURL)
	}
}

func (s *ParserSuite) TestFetchSmartRecruitersFromAPI_Happy() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"name": "Staff Software Engineer",
			"location": map[string]any{
				"city":    "Vancouver",
				"country": "CA",
				"remote":  true,
			},
			"typeOfEmployment": map[string]any{
				"typeId": "permanent",
				"label":  "Full-time",
			},
			"compensation": map[string]any{
				"min":      120000,
				"max":      180000,
				"currency": "CAD",
			},
			"jobAd": map[string]any{
				"sections": map[string]any{
					"jobDescription": map[string]any{
						"text": "<p>We are building observability tools.</p>",
					},
					"qualifications": map[string]any{
						"text": "<p>5+ years Go experience.</p>",
					},
				},
			},
		})
	}))
	defer mock.Close()

	job, err := parser.FetchSmartRecruitersFromAPI(
		context.Background(), mock.URL, "https://jobs.smartrecruiters.com/Grafana/123", "Grafana", "123",
	)
	s.Require().NoError(err)
	s.Require().NotNil(job)
	s.Equal("Staff Software Engineer", job.Title)
	s.Equal("SmartRecruiters", job.Source)
	s.Equal("Vancouver", job.Location)
	s.Contains(job.SalaryRaw, "120000")
	s.Contains(job.BodyMD, "observability tools")
}

func (s *ParserSuite) TestFetchSmartRecruitersFromAPI_NotFound() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mock.Close()

	_, err := parser.FetchSmartRecruitersFromAPI(
		context.Background(), mock.URL, "https://jobs.smartrecruiters.com/Acme/999", "Acme", "999",
	)
	s.Require().Error(err)
	s.Contains(err.Error(), "404")
}
