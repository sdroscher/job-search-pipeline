package feature_test

import (
	"net/http/httptest"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

func newServerWithStore(t *testing.T) (*httptest.Server, *db.Store) {
	t.Helper()
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	return ts, store
}
