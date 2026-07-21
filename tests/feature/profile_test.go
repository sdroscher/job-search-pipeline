package feature_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaleness_ArtifactFlaggedAfterProfileUpdate(t *testing.T) {
	ts, store := newServerWithStore(t)
	ctx := context.Background()

	today := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	_, err := store.CreateJob(ctx, db.CreateJobParams{
		ID:           "stale-test",
		Company:      "Acme",
		Role:         "SWE",
		Stage:        "Evaluated",
		Verdict:      "green",
		Added:        today,
		LastActivity: today,
	})
	require.NoError(t, err)

	putProfile(t, ts.URL, "# Resume v1")

	profile, err := store.GetProfile(ctx)
	require.NoError(t, err)

	_, err = store.CreateArtifact(ctx, db.CreateArtifactParams{
		JobID:       "stale-test",
		Type:        "resume",
		Filepath:    "./output/resume-acme-swe.md",
		ProfileHash: profile.ProfileHash,
	})
	require.NoError(t, err)

	// Update profile — should mark artifact stale via handlePutProfile.
	putProfile(t, ts.URL, "# Resume v2 with new content")

	artifacts, err := store.ListArtifacts(ctx, "stale-test")
	require.NoError(t, err)
	require.Len(t, artifacts, 1, "expected 1 artifact")
	assert.Equal(t, int64(1), artifacts[0].Stale, "artifact should be stale after profile update")
}

func TestBoard_ShowsStaleWarning(t *testing.T) {
	ts, store := newServerWithStore(t)
	ctx := context.Background()

	today := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	_, err := store.CreateJob(ctx, db.CreateJobParams{
		ID:           "stale-board",
		Company:      "Acme",
		Role:         "SWE",
		Stage:        "Evaluated",
		Verdict:      "green",
		Added:        today,
		LastActivity: today,
	})
	require.NoError(t, err)

	putProfile(t, ts.URL, "# Resume v1")

	profile, err := store.GetProfile(ctx)
	require.NoError(t, err)

	_, err = store.CreateArtifact(ctx, db.CreateArtifactParams{
		JobID:       "stale-board",
		Type:        "resume",
		Filepath:    "./output/resume.md",
		ProfileHash: profile.ProfileHash,
	})
	require.NoError(t, err)

	// Update profile — makes artifact stale.
	putProfile(t, ts.URL, "# Resume v2")

	resp, err := http.Get(ts.URL + "/") //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "⚠", "board should show stale warning for job with outdated artifact")
}

func putProfile(t *testing.T, baseURL, resume string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{"resume_md": resume})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, baseURL+"/api/profile", bytes.NewReader(body)) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
