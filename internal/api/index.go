package api

import (
	"context"
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

	staleJobs := s.staleJobSet(ctx, jobs)

	byStage := make(map[string][]db.Job)
	for _, job := range jobs {
		byStage[job.Stage] = append(byStage[job.Stage], job)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	renderErr := ui.IndexPage(byStage, staleJobs).Render(ctx, w)
	if renderErr != nil {
		log.Printf("render: %v", renderErr)
	}
}

func (s *Server) staleJobSet(ctx context.Context, jobs []db.Job) map[string]bool {
	stale := make(map[string]bool)

	for _, job := range jobs {
		artifacts, _ := s.store.ListArtifacts(ctx, job.ID)
		for _, artifact := range artifacts {
			if artifact.Stale == 1 {
				stale[job.ID] = true

				break
			}
		}
	}

	return stale
}
