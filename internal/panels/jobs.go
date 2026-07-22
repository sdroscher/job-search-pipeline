package panels

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/sdroscher/job-search-pipeline/internal/ui"
)

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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = ui.DetailPanel(job, activity).Render(r.Context(), w)
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

	staleJobs := h.staleJobSet(r.Context(), colJobs)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err = ui.Column(stage, colJobs, staleJobs).Render(r.Context(), w)
	if err != nil {
		log.Printf("render column: %v", err)
	}
}

// staleJobSet returns a set of job IDs that have at least one stale artifact.
func (h *JobPanelHandler) staleJobSet(ctx context.Context, jobs []db.Job) map[string]bool {
	stale := make(map[string]bool, len(jobs))

	for _, job := range jobs {
		artifacts, err := h.store.ListArtifacts(ctx, job.ID)
		if err != nil {
			log.Printf("list artifacts for stale check (job=%s): %v", job.ID, err)

			continue
		}

		for _, artifact := range artifacts {
			if artifact.Stale == 1 {
				stale[job.ID] = true

				break
			}
		}
	}

	return stale
}
