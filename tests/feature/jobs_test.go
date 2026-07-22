package feature_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobs(t *testing.T) {
	t.Run("getting a job that does not exist returns 404", func(t *testing.T) {
		ts := newTestServer(t)
		resp, err := http.Get(ts.URL + "/api/jobs/nonexistent") //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("after a job is created it can be retrieved and its fields match", func(t *testing.T) {
		ts := newTestServer(t)
		job := createJob(t, ts, "test-job", "TestCo", "Staff SWE", "Evaluated")
		assert.Equal(t, "TestCo", job["company"])
		assert.Equal(t, "Staff SWE", job["role"])
		assert.Equal(t, "Evaluated", job["stage"])
		resp, err := http.Get(ts.URL + "/api/jobs/test-job") //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("creating a job with a duplicate ID returns 409", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "dup-job", "DupCo", "SWE", "Evaluated")
		body, err := json.Marshal(map[string]any{
			"id": "dup-job", "company": "DupCo", "role": "SWE",
			"stage": "Evaluated", "verdict": "green",
		})
		require.NoError(t, err)
		resp, err := http.Post(ts.URL+"/api/jobs", "application/json", bytes.NewReader(body)) //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("deleting a job that does not exist returns 404", func(t *testing.T) {
		ts := newTestServer(t)
		req, err := http.NewRequestWithContext(
			context.Background(), http.MethodDelete, ts.URL+"/api/jobs/nonexistent", nil,
		)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("after stage is updated via HTMX panel response contains updated column HTML", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "stage-job", "TestCo", "Staff SWE", "Screening")
		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			ts.URL+"/panels/jobs/stage-job/stage",
			strings.NewReader("stage=Applied"),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(raw), "Applied", "response should contain updated column HTML")
	})
}
