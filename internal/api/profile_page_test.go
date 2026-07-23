package api_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/require"
)

func TestProfilePage_GetEmpty(t *testing.T) {
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `name="resume_md"`)
}

func TestProfilePage_GetWithProfile(t *testing.T) {
	store := db.NewTestStore(t)
	_, err := store.UpsertProfile(t.Context(), db.UpsertProfileParams{
		ResumeMd: "# My Resume", ProfileHash: "abc",
	})
	require.NoError(t, err)

	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "# My Resume")
}

const testSalaryMin = 100000

func TestProfilePage_Post(t *testing.T) {
	store := db.NewTestStore(t)
	srv := api.NewServer(store, api.Config{OutputDir: t.TempDir()})

	form := url.Values{}
	form.Set("resume_md", "# Updated Resume")
	form.Set("salary_min", strconv.Itoa(testSalaryMin))
	form.Set("remote_pref", "remote-only")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Equal(t, "/profile?saved=1", w.Header().Get("Location"))

	saved, err := store.GetProfile(t.Context())
	require.NoError(t, err)
	require.Equal(t, "# Updated Resume", saved.ResumeMd)
	require.NotNil(t, saved.SalaryMin)
	require.Equal(t, int64(testSalaryMin), *saved.SalaryMin)
}
