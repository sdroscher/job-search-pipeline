package feature_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	return ts
}

func TestBoard_EmptyRendersAllColumns(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/") //nolint:noctx
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want 200", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	body := string(raw)

	for _, stage := range []string{"Evaluated", "Applied", "Screening", "Interviewing", "Final Round", "Offer"} {
		if !strings.Contains(body, stage) {
			t.Errorf("board missing column %q", stage)
		}
	}
}
