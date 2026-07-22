package feature_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivity(t *testing.T) {
	t.Run("creating an activity for a job that does not exist returns 404", func(t *testing.T) {
		ts := newTestServer(t)

		req, err := http.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			ts.URL+"/api/jobs/nonexistent/activity",
			strings.NewReader(`{"action":"Applied"}`),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("after an activity is logged it appears in the job detail panel", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "act-job", "ActivityCo", "SWE", "Evaluated")
		createActivity(t, ts, "act-job", "Interviewed")

		panel := getDetailPanel(t, ts, "act-job")

		assert.Contains(t, panel, "Interviewed", "detail panel should contain the logged activity")
	})
}
