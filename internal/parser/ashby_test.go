package parser_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
)

func (s *ParserSuite) TestFetchAshby() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"jobs": []map[string]any{
				{
					"id":              "abc-123",
					"title":           "Staff Software Engineer",
					"descriptionHtml": "<p>We are building great things.</p>",
					"locationName":    "Remote, US",
				},
			},
		})
	}))
	defer mock.Close()

	job, err := parser.FetchAshbyFromAPI(context.Background(), mock.URL, "https://jobs.ashbyhq.com/acme/abc-123", "acme", "abc-123")
	s.Require().NoError(err)
	s.Require().NotNil(job)
	s.Equal("Staff Software Engineer", job.Title)
	s.Equal("Ashby", job.Source)
	s.Equal("Remote, US", job.Location)
}

func (s *ParserSuite) TestFetchAshbyFromAPI_NotFound() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"jobs": []any{}}) //nolint:errcheck
	}))
	defer mock.Close()

	_, err := parser.FetchAshbyFromAPI(context.Background(), mock.URL, "https://jobs.ashbyhq.com/acme/abc-123", "acme", "abc-123")
	s.Require().Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *ParserSuite) TestFetchAshbyFromAPI_Unauthorized() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mock.Close()

	_, err := parser.FetchAshbyFromAPI(context.Background(), mock.URL, "https://jobs.ashbyhq.com/acme/abc-123", "acme", "abc-123")
	s.Require().Error(err)
	s.Contains(err.Error(), "401")
}

func (s *ParserSuite) TestDetectATS_Ashby() {
	urls := []string{
		"https://jobs.ashbyhq.com/temporal/abc-123",
		"https://jobs.ashbyhq.com/acme/def-456",
	}
	for _, rawURL := range urls {
		s.Equal(parser.ATSAshby, parser.DetectATS(rawURL), "URL: %s", rawURL)
	}
}
