package api

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

type createActivityRequest struct {
	Date   string `json:"date"`
	Action string `json:"action"`
	Notes  string `json:"notes"`
}

func (s *Server) handleCreateActivity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	_, jobErr := s.store.GetJob(r.Context(), id)
	if errors.Is(jobErr, sql.ErrNoRows) {
		http.Error(w, "job not found", http.StatusNotFound)

		return
	}

	if jobErr != nil {
		http.Error(w, jobErr.Error(), http.StatusInternalServerError)

		return
	}

	var req createActivityRequest

	err := readJSON(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if req.Action == "" {
		http.Error(w, "action required", http.StatusBadRequest)

		return
	}

	date, err := parseDate(req.Date)
	if err != nil {
		http.Error(w, "invalid date: "+err.Error(), http.StatusBadRequest)

		return
	}

	var notes *string
	if req.Notes != "" {
		notes = &req.Notes
	}

	entry, err := s.store.CreateActivityEntry(r.Context(), db.CreateActivityEntryParams{
		JobID:  id,
		Date:   date,
		Action: req.Action,
		Notes:  notes,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	s.touchJob(r.Context(), id, date)

	writeJSON(w, http.StatusCreated, entry)
}

// touchJob moves a job's last_activity to date. Logging activity is what makes
// that date move; UpdateJob leaves it alone so corrections don't make a stale
// application look freshly worked. Using the entry's own date means a
// backdated entry doesn't claim the job was touched today. Best-effort: the
// activity entry itself has already been written.
func (s *Server) touchJob(ctx context.Context, jobID string, date time.Time) {
	err := s.store.TouchJobActivity(ctx, db.TouchJobActivityParams{ID: jobID, LastActivity: date})
	if err != nil {
		log.Printf("touch last_activity failed: %v (id=%q)", err, jobID) //nolint:gosec
	}
}
