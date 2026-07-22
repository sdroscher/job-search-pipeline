package feature_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifacts(t *testing.T) {
	t.Run("listing artifacts for a job that does not exist returns 404", func(t *testing.T) {
		ts := newTestServer(t)

		resp, err := http.Get(ts.URL + "/api/jobs/nonexistent/artifacts") //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("creating an artifact with filepath outside output directory returns 400", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "art-path-job", "ArtCo", "SWE", "Evaluated")

		body, err := json.Marshal(map[string]any{
			"type":         "resume",
			"filepath":     "/etc/evil",
			"profile_hash": "abc123",
		})
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			ts.URL+"/api/jobs/art-path-job/artifacts",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("after an artifact is created it appears in the artifact list for the job", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "art-list-job", "ListCo", "SWE", "Evaluated")

		profile := putProfile(t, ts, "# My Resume")
		hash, _ := profile["profile_hash"].(string)

		createArtifact(t, ts, "art-list-job", "resume", "resume-list.md", hash)

		resp, err := http.Get(ts.URL + "/api/jobs/art-list-job/artifacts") //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var artifacts []map[string]any
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&artifacts))
		require.Len(t, artifacts, 1)
		assert.Equal(t, "resume", artifacts[0]["type"])
	})
}
