package parser_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
)

func (s *ParserSuite) TestFetchLever() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"text":        "Principal Engineer",
			"categories":  map[string]any{"location": "Remote", "team": "Platform"},
			"description": "<p>Join our team.</p>",
			"lists": []any{
				map[string]any{"text": "Responsibilities", "content": "<li>Build things</li>"},
			},
		})
	}))
	defer mock.Close()

	job, err := parser.FetchLeverFromAPI(mock.URL, "https://jobs.lever.co/acme/abc-123")
	s.Require().NoError(err)
	s.Require().NotNil(job)
	s.Equal("Principal Engineer", job.Title)
	s.Equal("Lever", job.Source)
	s.Equal("Remote", job.Location)
	s.Equal("Platform", job.Department)
	s.Contains(job.BodyMD, "Responsibilities")
}

func (s *ParserSuite) TestFetchLeverFromAPI_Non200() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mock.Close()

	_, err := parser.FetchLeverFromAPI(mock.URL, "https://jobs.lever.co/acme/abc-123")
	s.Require().Error(err)
}

func (s *ParserSuite) TestDetectATS_Lever() {
	urls := []string{
		"https://jobs.lever.co/temporal/abc-123",
		"https://jobs.lever.co/acme/def-456",
	}
	for _, rawURL := range urls {
		s.Equal(parser.ATSLever, parser.DetectATS(rawURL), "URL: %s", rawURL)
	}
}
