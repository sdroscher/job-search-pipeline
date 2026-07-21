package parser_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
)

func (s *ParserSuite) TestFetchGreenhouse() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"title":        "Senior Software Engineer",
			"absolute_url": "https://boards.greenhouse.io/acme/jobs/123",
			"content":      "<p>We are looking for a Senior SWE to join our team.</p>",
			"location":     map[string]any{"name": "Remote"},
			"departments":  []any{map[string]any{"name": "Engineering"}},
			"metadata":     []any{},
		})
	}))
	defer mock.Close()

	job, err := parser.FetchGreenhouseFromAPI(
		mock.URL+"/v1/boards/acme/jobs/123",
		"https://boards.greenhouse.io/acme/jobs/123",
	)
	s.Require().NoError(err)
	s.Require().NotNil(job)
	s.Equal("Senior Software Engineer", job.Title)
	s.Equal("Greenhouse", job.Source)
	s.Equal("Remote", job.Location)
	s.Equal("Engineering", job.Department)
	s.Contains(job.BodyMD, "Senior SWE")
}

func (s *ParserSuite) TestFetchGreenhouseFromAPI_Non200() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer mock.Close()

	_, err := parser.FetchGreenhouseFromAPI(mock.URL, "https://boards.greenhouse.io/acme/jobs/1")
	s.Require().Error(err)
}

func (s *ParserSuite) TestFetchGreenhouse_URLMismatch() {
	_, err := parser.FetchGreenhouse("https://not-a-greenhouse-url.com/jobs/123")
	s.Require().Error(err)
}

func (s *ParserSuite) TestDetectATS_Greenhouse() {
	urls := []string{
		"https://boards.greenhouse.io/temporal/jobs/12345",
		"https://boards.greenhouse.io/chainguard/jobs/99",
	}
	for _, rawURL := range urls {
		s.Equal(parser.ATSGreenhouse, parser.DetectATS(rawURL), "URL: %s", rawURL)
	}
}
