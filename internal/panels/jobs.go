package panels

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/sdroscher/job-search-pipeline/internal/ui"
)

func parseStrings(s *string) []string {
	if s == nil {
		return nil
	}

	var out []string
	_ = json.Unmarshal([]byte(*s), &out)

	return out
}

func parseCompanyValues(s *string) []ui.CompanyValue {
	if s == nil {
		return nil
	}

	var out []ui.CompanyValue
	_ = json.Unmarshal([]byte(*s), &out)

	return out
}

// JobPanelHandler handles HTMX panel requests for individual jobs.
type JobPanelHandler struct {
	store *db.Store
}

// NewJobPanelHandler creates a new JobPanelHandler.
func NewJobPanelHandler(store *db.Store) *JobPanelHandler {
	return &JobPanelHandler{store: store}
}

// HandleDetail renders the job detail panel fragment.
func (h *JobPanelHandler) HandleDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	job, err := h.store.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	activity, err := h.store.ListActivityLog(r.Context(), id)
	if err != nil {
		log.Printf("list activity log failed: %v (id=%q)", err, id) //nolint:gosec
		activity = []db.ActivityLog{}
	}

	artifacts, err := h.store.ListArtifacts(r.Context(), id)
	if err != nil {
		log.Printf("list artifacts failed: %v (id=%q)", err, id) //nolint:gosec
		artifacts = []db.Artifact{}
	}

	data := ui.DetailData{
		Job:           job,
		Activity:      activity,
		Positives:     parseStrings(job.Positives),
		Concerns:      parseStrings(job.Concerns),
		CompanyValues: parseCompanyValues(job.CompanyValues),
		Artifacts:     artifacts,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = ui.DetailPanel(data).Render(r.Context(), w)
	if err != nil {
		log.Printf("render detail panel: %v", err)
	}
}

// HandleUpdateStage updates the stage of a job via a form POST.
func (h *JobPanelHandler) HandleUpdateStage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	stage := r.FormValue("stage")
	if stage == "" {
		http.Error(w, "stage required", http.StatusBadRequest)

		return
	}

	_, err = h.store.UpdateJob(r.Context(), db.UpdateJobParams{ID: id, Stage: &stage})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	// Return updated column HTML so HTMX can swap it in.
	jobs, err := h.store.ListJobs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	colJobs := make([]db.Job, 0)
	for _, job := range jobs {
		if job.Stage == stage {
			colJobs = append(colJobs, job)
		}
	}

	staleJobs := h.store.StaleJobSet(r.Context(), colJobs)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = ui.Column(stage, colJobs, staleJobs).Render(r.Context(), w)
	if err != nil {
		log.Printf("render column: %v", err)
	}
}
