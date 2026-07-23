package feature_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfile(t *testing.T) {
	t.Run("getting a profile when none exists returns 404", func(t *testing.T) {
		ts := newTestServer(t)

		resp, err := http.Get(ts.URL + "/api/profile") //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("after a profile is saved it can be retrieved and includes a profile_hash", func(t *testing.T) {
		ts := newTestServer(t)
		putProfile(t, ts, "# My Resume")

		profile := getProfile(t, ts)

		assert.Equal(t, "# My Resume", profile["resume_md"])
		assert.NotEmpty(t, profile["profile_hash"])
	})

	t.Run("after a profile update artifacts for existing jobs are marked stale", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "stale-profile-job", "StaleCo", "SWE", "Evaluated")

		profile := putProfile(t, ts, "# Resume v1")
		hash, ok := profile["profile_hash"].(string)
		require.True(t, ok, "profile_hash must be a string")

		createArtifact(t, ts, "stale-profile-job", "resume", "resume-stale.md", hash)

		// Update profile — should mark the artifact stale.
		putProfile(t, ts, "# Resume v2 with new content")

		resp, err := http.Get(ts.URL + "/api/jobs/stale-profile-job/artifacts") //nolint:noctx
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		// JSON encodes the stale integer column as a number; check the raw response.
		assert.Contains(t, string(raw), `"stale":1`, "artifact should be stale after profile update")
	})
}
