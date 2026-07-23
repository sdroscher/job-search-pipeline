package api

import (
	"log"
	"net/http"

	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/sdroscher/job-search-pipeline/internal/ui"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	jobs, err := s.store.ListJobs(ctx)
	if err != nil {
		http.Error(w, "failed to load jobs", http.StatusInternalServerError)

		return
	}

	closedJobs, closedErr := s.store.ListClosedJobs(ctx)
	if closedErr != nil {
		log.Printf("list closed jobs: %v", closedErr)
		closedJobs = []db.Job{}
	}

	staleJobs := s.store.StaleJobSet(ctx, jobs)

	byStage := make(map[string][]db.Job)
	for _, job := range jobs {
		byStage[job.Stage] = append(byStage[job.Stage], job)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	renderErr := ui.IndexPage(byStage, closedJobs, staleJobs).Render(ctx, w)
	if renderErr != nil {
		log.Printf("render: %v", renderErr)
	}
}
