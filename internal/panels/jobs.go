package panels

import (
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
		http.Error(w, "not found", http.StatusNotFound)

		return
	}

	activity, err := h.store.ListActivityLog(r.Context(), id)
	if err != nil {
		log.Printf("list activity log failed: %v (id=%q)", err, id) // #nosec G706
		activity = []db.ActivityLog{}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = ui.DetailPanel(job, activity).Render(r.Context(), w)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
}
