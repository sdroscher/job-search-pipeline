package api

import (
	"database/sql"
	"errors"
	"net/http"

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

	writeJSON(w, http.StatusCreated, entry)
}
