package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/require"
)

func TestDetailPanel_ShowsJobID(t *testing.T) {
	store := db.NewTestStore(t)
	_, err := store.CreateJob(t.Context(), db.CreateJobParams{
		ID:      "acme-staff-swe",
		Company: "Acme",
		Role:    "Staff SWE",
		Stage:   "Evaluated",
		Verdict: "green",
		Added:   time.Now(),
	})
	require.NoError(t, err)

	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/panels/jobs/acme-staff-swe", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "acme-staff-swe")
	require.Contains(t, w.Body.String(), "copy-id-btn")
}
