package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServer(t *testing.T) (*httptest.Server, *db.Store) {
	t.Helper()
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	return ts, store
}

func TestListJobs_Empty(t *testing.T) {
	ts, _ := newServer(t)

	resp, err := http.Get(ts.URL + "/api/jobs") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jobs []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&jobs))
	assert.Empty(t, jobs)
}

func TestCreateJob_ThenGet(t *testing.T) {
	ts, store := newServer(t)
	today := time.Now().UTC().Format("2006-01-02")

	body, err := json.Marshal(map[string]any{
		"id":            "acme-staff-swe",
		"company":       "Acme",
		"role":          "Staff SWE",
		"stage":         "Evaluated",
		"verdict":       "green",
		"fit_score":     9,
		"added":         today,
		"last_activity": today,
	})
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/api/jobs", "application/json", bytes.NewReader(body)) //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	resp2, err := http.Get(ts.URL + "/api/jobs/acme-staff-swe") //nolint:noctx
	require.NoError(t, err)

	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var job map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&job))
	assert.Equal(t, "Acme", job["company"])
	assert.Equal(t, "Staff SWE", job["role"])

	// Verify activity log entry was created with Action: "Added"
	entries, err := store.ListActivityLog(context.Background(), "acme-staff-swe")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Added", entries[0].Action)
}

func TestUpdateJob_Stage(t *testing.T) {
	ts, store := newServer(t)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	_, err := store.CreateJob(ctx, db.CreateJobParams{
		ID:           "job-1",
		Company:      "X",
		Role:         "SWE",
		Stage:        "Evaluated",
		Verdict:      "yellow",
		Added:        today,
		LastActivity: today,
	})
	require.NoError(t, err)

	body, err := json.Marshal(map[string]any{"stage": "Applied"})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, ts.URL+"/api/jobs/job-1", bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var job map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&job))
	assert.Equal(t, "Applied", job["stage"])
}

func TestDeleteJob_SoftDelete(t *testing.T) {
	ts, store := newServer(t)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	_, err := store.CreateJob(ctx, db.CreateJobParams{
		ID:           "job-del",
		Company:      "Del Corp",
		Role:         "Dev",
		Stage:        "Evaluated",
		Verdict:      "red",
		Added:        today,
		LastActivity: today,
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, ts.URL+"/api/jobs/job-del", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify soft-delete: job still exists but stage is "Won't Apply"
	job, err := store.GetJob(ctx, "job-del")
	require.NoError(t, err)
	assert.Equal(t, "Won't Apply", job.Stage)
}

func TestGetJob_NotFound(t *testing.T) {
	ts, _ := newServer(t)

	resp, err := http.Get(ts.URL + "/api/jobs/nonexistent") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestUpdateJob_NotFound(t *testing.T) {
	ts, _ := newServer(t)

	body, err := json.Marshal(map[string]any{"stage": "Applied"})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPatch, ts.URL+"/api/jobs/nonexistent", bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateActivity_UnknownJob(t *testing.T) {
	ts, _ := newServer(t)

	body, err := json.Marshal(map[string]any{
		"date":   time.Now().UTC().Format("2006-01-02"),
		"action": "Applied",
		"notes":  "",
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/jobs/no-such-job/activity", bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDeleteJob_NotFound(t *testing.T) {
	ts, _ := newServer(t)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, ts.URL+"/api/jobs/nonexistent", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestListArtifacts_UnknownJob(t *testing.T) {
	ts, _ := newServer(t)

	resp, err := http.Get(ts.URL + "/api/jobs/nonexistent/artifacts") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateArtifact_PathTraversal(t *testing.T) {
	ts, store := newServer(t)
	ctx := context.Background()
	today := time.Now().UTC().Truncate(24 * time.Hour)

	_, err := store.CreateJob(ctx, db.CreateJobParams{
		ID:           "job-path",
		Company:      "Acme",
		Role:         "SWE",
		Stage:        "Evaluated",
		Verdict:      "green",
		Added:        today,
		LastActivity: today,
	})
	require.NoError(t, err)

	body, err := json.Marshal(map[string]any{
		"type":         "cover_letter",
		"filepath":     "/etc/evil",
		"profile_hash": "abc123",
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/api/jobs/job-path/artifacts", bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCreateArtifact_UnknownJob(t *testing.T) {
	ts, _ := newServer(t)

	body, err := json.Marshal(map[string]any{
		"type":         "cover_letter",
		"filepath":     t.TempDir() + "/cover.md",
		"profile_hash": "abc123",
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/jobs/no-such-job/artifacts", bytes.NewReader(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
