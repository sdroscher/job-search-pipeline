package panels

import (
	"context"
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

// isValidCloseStage checks if a stage is a valid close target.
// Valid close stages are terminal states or the reopen target.
func isValidCloseStage(stage string) bool {
	switch stage {
	case "Rejected", "Listing Withdrawn", "Declined", "Won't Apply", "Evaluated":
		return true
	}

	return false
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

// HandleCloseJob transitions a job to/from a closed stage and returns HTMX OOB swaps.
func (h *JobPanelHandler) HandleCloseJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	parseErr := r.ParseForm()
	if parseErr != nil {
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	newStage := r.FormValue("stage")
	fromStage := r.FormValue("from_stage")

	if newStage == "" || fromStage == "" {
		http.Error(w, "stage and from_stage required", http.StatusBadRequest)

		return
	}

	if !isValidCloseStage(newStage) {
		http.Error(w, "invalid stage", http.StatusBadRequest)

		return
	}

	ctx := r.Context()

	_, err := h.store.UpdateJob(ctx, db.UpdateJobParams{ID: jobID, Stage: &newStage})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	colData, err := h.buildCloseJobData(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	fromJobs := jobsForStage(colData.active, fromStage)
	toJobs := jobsForStage(colData.active, newStage)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	renderErr := ui.CloseJobResponse(fromStage, fromJobs, newStage, toJobs, colData.closed, colData.stale).Render(ctx, w)
	if renderErr != nil {
		log.Printf("render close job OOB: %v", renderErr)
	}
}

type closeJobData struct {
	active []db.Job
	closed []db.Job
	stale  map[string]bool
}

func (h *JobPanelHandler) buildCloseJobData(ctx context.Context) (*closeJobData, error) {
	activeJobs, err := h.store.ListJobs(ctx)
	if err != nil {
		return nil, err
	}

	closedJobs, err := h.store.ListClosedJobs(ctx)
	if err != nil {
		return nil, err
	}

	return &closeJobData{
		active: activeJobs,
		closed: closedJobs,
		stale:  h.store.StaleJobSet(ctx, activeJobs),
	}, nil
}

func jobsForStage(jobs []db.Job, stage string) []db.Job {
	result := make([]db.Job, 0)
	for _, job := range jobs {
		if job.Stage == stage {
			result = append(result, job)
		}
	}

	return result
}
