package feature_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/require"
)

const contentTypeJSON = "application/json"

type testServer struct {
	*httptest.Server
	outputDir string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	outputDir := t.TempDir()
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: outputDir})
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	return &testServer{Server: ts, outputDir: outputDir}
}

// createJob POSTs to /api/jobs and returns the decoded job map.
func createJob(t *testing.T, ts *testServer, id, company, role, stage string) map[string]any {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"id": id, "company": company, "role": role, "stage": stage, "verdict": "green",
	})
	require.NoError(t, err)

	resp, err := http.Post(ts.URL+"/api/jobs", contentTypeJSON, bytes.NewReader(body)) //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var job map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&job))

	return job
}

// putProfile PUTs to /api/profile and returns the decoded profile map (includes profile_hash).
func putProfile(t *testing.T, ts *testServer, resumeMD string) map[string]any {
	t.Helper()

	body, err := json.Marshal(map[string]any{"resume_md": resumeMD})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/profile", bytes.NewReader(body)) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentTypeJSON)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var profile map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&profile))

	return profile
}

// getProfile GETs /api/profile and returns the decoded map.
func getProfile(t *testing.T, ts *testServer) map[string]any {
	t.Helper()

	resp, err := http.Get(ts.URL + "/api/profile") //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var profile map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&profile))

	return profile
}

// createArtifact POSTs to /api/jobs/{id}/artifacts with filepath under ts.outputDir.
func createArtifact(t *testing.T, ts *testServer, jobID, artifactType, filename, profileHash string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{
		"type":         artifactType,
		"filepath":     filepath.Join(ts.outputDir, filename),
		"profile_hash": profileHash,
	})
	require.NoError(t, err)

	req, err := http.NewRequest( //nolint:noctx
		http.MethodPost,
		fmt.Sprintf("%s/api/jobs/%s/artifacts", ts.URL, jobID),
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentTypeJSON)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

// createActivity POSTs to /api/jobs/{id}/activity.
func createActivity(t *testing.T, ts *testServer, jobID, action string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{"action": action})
	require.NoError(t, err)

	req, err := http.NewRequest( //nolint:noctx
		http.MethodPost,
		fmt.Sprintf("%s/api/jobs/%s/activity", ts.URL, jobID),
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", contentTypeJSON)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

// getBoard GETs / and returns the body as a string.
func getBoard(t *testing.T, ts *testServer) string {
	t.Helper()

	resp, err := http.Get(ts.URL + "/") //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(raw)
}

// getBoardPanel GETs /panels/board and returns the body as a string.
func getBoardPanel(t *testing.T, ts *testServer) string {
	t.Helper()

	resp, err := http.Get(ts.URL + "/panels/board") //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(raw)
}

// getDetailPanel GETs /panels/jobs/{id} and returns the body as a string.
func getDetailPanel(t *testing.T, ts *testServer, jobID string) string {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("%s/panels/jobs/%s", ts.URL, jobID)) //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return string(raw)
}
