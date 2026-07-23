package parser_test

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
)

func (s *ParserSuite) TestDetectATS_BambooHR() {
	urls := []string{
		"https://grafana.bamboohr.com/careers/123",
		"https://acme.bamboohr.com/jobs/456",
	}
	for _, rawURL := range urls {
		s.Equal(parser.ATSBambooHR, parser.DetectATS(rawURL), "URL: %s", rawURL)
	}
}

func (s *ParserSuite) TestFetchBambooHRFromURL_Happy() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
  <meta property="og:title" content="Staff Software Engineer"/>
</head>
<body>
  <h1>Staff Software Engineer</h1>
  <p>We are building something great.</p>
</body>
</html>`))
	}))
	defer mock.Close()

	job, err := parser.FetchBambooHRFromURL(context.Background(), mock.URL, "https://acme.bamboohr.com/careers/123")
	s.Require().NoError(err)
	s.Require().NotNil(job)
	s.Equal("Staff Software Engineer", job.Title)
	s.Equal("BambooHR", job.Source)
	s.Equal("https://acme.bamboohr.com/careers/123", job.SourceURL)
	s.Contains(job.BodyMD, "building something great")
}

func (s *ParserSuite) TestFetchBambooHRFromURL_NonOK() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mock.Close()

	_, err := parser.FetchBambooHRFromURL(context.Background(), mock.URL, "https://acme.bamboohr.com/careers/999")
	s.Require().Error(err)
	s.Contains(err.Error(), "404")
}
