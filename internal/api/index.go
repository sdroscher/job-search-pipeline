package api

import (
	"log"
	"net/http"

	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/sdroscher/job-search-pipeline/internal/ui"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ListJobs(r.Context())
	if err != nil {
		http.Error(w, "failed to load jobs", http.StatusInternalServerError)

		return
	}

	byStage := make(map[string][]db.Job)
	for _, job := range jobs {
		byStage[job.Stage] = append(byStage[job.Stage], job)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	renderErr := ui.IndexPage(byStage).Render(r.Context(), w)
	if renderErr != nil {
		log.Printf("render: %v", renderErr)
	}
}
