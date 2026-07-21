package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
