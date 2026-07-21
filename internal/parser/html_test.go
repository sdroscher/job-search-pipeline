package parser_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
)

func (s *ParserSuite) TestScrapeHTML() {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w,
			`<html><head><title>Staff SWE at Acme</title></head>`+
				`<body><h1>Staff Software Engineer</h1><p>We are Acme Corp. Join us.</p></body></html>`,
		)
	}))
	defer mock.Close()

	job, err := parser.ScrapeHTMLFromURL(mock.URL, mock.URL)
	s.Require().NoError(err)
	s.Require().NotNil(job)
	s.Contains(job.BodyMD, "Acme Corp")
	s.Equal("HTML", job.Source)
	s.Equal(mock.URL, job.SourceURL)
}

func (s *ParserSuite) TestDetectATS_HTML() {
	urls := []string{
		"https://example.com/careers/job-123",
		"https://company.com/jobs/senior-swe",
	}
	for _, rawURL := range urls {
		s.Equal(parser.ATSHTML, parser.DetectATS(rawURL), "URL: %s", rawURL)
	}
}
