package feature_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedJob(t *testing.T, store *db.Store) db.Job {
	t.Helper()
	today := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	job, err := store.CreateJob(context.Background(), db.CreateJobParams{
		ID:           "test-job",
		Company:      "TestCo",
		Role:         "Staff SWE",
		Stage:        "Evaluated",
		Verdict:      "green",
		Added:        today,
		LastActivity: today,
	})
	require.NoError(t, err)

	return job
}

func TestDetailPanel_RendersJob(t *testing.T) {
	ts, store := newServerWithStore(t)
	seedJob(t, store)

	resp, err := http.Get(ts.URL + "/panels/jobs/test-job") //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Contains(t, string(raw), "TestCo", "detail panel missing company name")
}

func TestUpdateStage_ViaHTMX(t *testing.T) {
	ts, store := newServerWithStore(t)
	seedJob(t, store)

	body := strings.NewReader("stage=Applied")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/panels/jobs/test-job/stage", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify DB was updated
	job, err := store.GetJob(context.Background(), "test-job")
	require.NoError(t, err)
	assert.Equal(t, "Applied", job.Stage, "stage not updated")
}
