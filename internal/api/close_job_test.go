package api_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedJob(t *testing.T, store *db.Store, id, stage string) db.Job {
	t.Helper()

	job, err := store.CreateJob(t.Context(), db.CreateJobParams{
		ID:      id,
		Company: "Acme",
		Role:    "SWE",
		Stage:   stage,
		Verdict: "green",
		Added:   time.Now(),
	})
	require.NoError(t, err)

	return job
}

func TestCloseJob_RemovesFromActiveBoard(t *testing.T) {
	store := db.NewTestStore(t)
	seedJob(t, store, "acme-swe", "Applied")

	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})

	form := url.Values{}
	form.Set("stage", "Rejected")
	form.Set("from_stage", "Applied")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/panels/jobs/acme-swe/close",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	// OOB clears the detail panel
	assert.Contains(t, body, `id="detail-panel"`)
	// OOB contains the updated "Applied" column (now empty)
	assert.Contains(t, body, `data-stage="Applied"`)
	// OOB contains the closed section showing the job
	assert.Contains(t, body, "closed-section")

	// DB: job stage is now Rejected
	job, err := store.GetJob(t.Context(), "acme-swe")
	require.NoError(t, err)
	assert.Equal(t, "Rejected", job.Stage)
}

func TestCloseJob_Reopen(t *testing.T) {
	store := db.NewTestStore(t)
	seedJob(t, store, "acme-swe", "Rejected")

	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})

	form := url.Values{}
	form.Set("stage", "Evaluated")
	form.Set("from_stage", "Rejected")
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/panels/jobs/acme-swe/close",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()
	assert.Contains(t, body, `id="detail-panel"`)
	// OOB has Evaluated column
	assert.Contains(t, body, `data-stage="Evaluated"`)

	job, err := store.GetJob(t.Context(), "acme-swe")
	require.NoError(t, err)
	assert.Equal(t, "Evaluated", job.Stage)
}

func TestListJobs_ExcludesClosedStages(t *testing.T) {
	store := db.NewTestStore(t)
	seedJob(t, store, "active-job", "Applied")
	seedJob(t, store, "closed-job", "Rejected")

	jobs, err := store.ListJobs(t.Context())
	require.NoError(t, err)

	for _, job := range jobs {
		assert.NotEqual(t, "closed-job", job.ID)
	}
}

func TestListClosedJobs(t *testing.T) {
	store := db.NewTestStore(t)
	seedJob(t, store, "active-job", "Applied")
	seedJob(t, store, "rejected-job", "Rejected")
	seedJob(t, store, "declined-job", "Declined")

	closed, err := store.ListClosedJobs(t.Context())
	require.NoError(t, err)

	ids := make(map[string]bool)
	for _, job := range closed {
		ids[job.ID] = true
	}

	assert.True(t, ids["rejected-job"])
	assert.True(t, ids["declined-job"])
	assert.False(t, ids["active-job"])
}
