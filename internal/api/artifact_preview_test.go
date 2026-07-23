package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/require"
)

const filePerms = 0o600

func TestArtifactPreview(t *testing.T) {
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})

	// Create a job and an artifact entry pointing to a temp file.
	ctx := t.Context()
	job, err := store.CreateJob(ctx, db.CreateJobParams{
		ID: "test-job", Company: "Acme", Role: "SWE", Stage: "Applied", Verdict: "green",
	})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	content := "# Test Resume\n\nSome content here."
	fPath := filepath.Join(tmpDir, "resume-acme-swe.md")
	require.NoError(t, os.WriteFile(fPath, []byte(content), filePerms))

	artifact, err := store.CreateArtifact(ctx, db.CreateArtifactParams{
		JobID: job.ID, Type: "resume", Filepath: fPath, ProfileHash: "abc123",
	})
	require.NoError(t, err)

	path := fmt.Sprintf("/panels/jobs/%s/artifacts/%d", job.ID, artifact.ID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	w := httptest.NewRecorder()

	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Test Resume")
	require.Contains(t, w.Body.String(), "markdown-body")
}
