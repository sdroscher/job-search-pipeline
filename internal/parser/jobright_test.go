package parser_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJobrightURL = "https://jobright.ai/jobs/info/abc123"

func TestDetectATS_Jobright(t *testing.T) {
	assert.Equal(t, parser.ATSJobright, parser.DetectATS(testJobrightURL))
}

func TestFetchJobrightFromURL_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head>
			<title>Staff Engineer - Widgets Inc | Jobright</title>
			<meta property="og:title" content="Staff Engineer - Widgets Inc"/>
			<meta name="description" content="Widgets Inc is hiring a Staff Engineer."/>
		</head><body>
			<h1>Staff Engineer</h1>
			<p class="company-name">Widgets Inc</p>
			<div id="job-description"><p>Come build great things.</p></div>
		</body></html>`))
	}))
	defer srv.Close()

	job, err := parser.FetchJobrightFromURL(context.Background(), srv.URL, testJobrightURL)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "Staff Engineer", job.Title)
	assert.Equal(t, "Widgets Inc", job.Company)
	assert.Equal(t, "Jobright", job.Source)
	assert.Equal(t, testJobrightURL, job.SourceURL)
}

func TestFetchJobrightFromURL_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	job, err := parser.FetchJobrightFromURL(context.Background(), srv.URL, testJobrightURL)
	assert.Nil(t, job)
	assert.ErrorIs(t, err, parser.ErrJobrightHTTPStatus)
}
