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

func TestDetectATS_Jobgether(t *testing.T) {
	assert.Equal(t, parser.ATSJobgether, parser.DetectATS("https://jobgether.com/offer/abc123"))
	assert.Equal(t, parser.ATSJobgether, parser.DetectATS("https://jobgether.com/en/jobs/456"))
}

func TestFetchJobgetherFromURL_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head>
			<title>Senior Backend Engineer at Acme Corp | Jobgether</title>
			<meta property="og:title" content="Senior Backend Engineer at Acme Corp"/>
		</head><body>
			<h1>Senior Backend Engineer at Acme Corp</h1>
			<div class="job-description"><p>We are looking for a senior engineer.</p></div>
		</body></html>`))
	}))
	defer srv.Close()

	job, err := parser.FetchJobgetherFromURL(context.Background(), srv.URL, "https://jobgether.com/offer/abc123")
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, "Senior Backend Engineer at Acme Corp", job.Title)
	assert.Equal(t, "Acme Corp", job.Company)
	assert.Equal(t, "Jobgether", job.Source)
	assert.Equal(t, "https://jobgether.com/offer/abc123", job.SourceURL)
	assert.NotEmpty(t, job.BodyMD)
}

func TestFetchJobgetherFromURL_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	job, err := parser.FetchJobgetherFromURL(context.Background(), srv.URL, "https://jobgether.com/offer/abc123")
	assert.Nil(t, job)
	assert.ErrorIs(t, err, parser.ErrJobgetherHTTPStatus)
}
