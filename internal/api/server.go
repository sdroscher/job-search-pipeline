package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/sdroscher/job-search-pipeline/internal/panels"
	"github.com/sdroscher/job-search-pipeline/internal/ui"
)

const routeRoot = "/"

// Config holds API-layer configuration.
type Config struct {
	OutputDir string
}

// Server holds dependencies for all HTTP handlers.
type Server struct {
	store         *db.Store
	config        Config
	jobPanel      *panels.JobPanelHandler
	artifactPanel *panels.ArtifactPanelHandler
}

func NewServer(store *db.Store, cfg Config) *Server {
	return &Server{
		store:         store,
		config:        cfg,
		jobPanel:      panels.NewJobPanelHandler(store),
		artifactPanel: panels.NewArtifactPanelHandler(store),
	}
}

// Router returns the fully-wired chi router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// JSON API
	r.Route("/api", func(r chi.Router) {
		r.Route("/profile", s.profileRoutes)
		r.Route("/jobs", s.jobRoutes)
		r.Post("/parse", s.handleParse)
	})

	// HTMX panel fragments
	r.Route("/panels", func(r chi.Router) {
		r.Get("/board", s.handleBoardPanel)
		r.Get("/jobs/{id}", s.handleJobDetailPanel)
		r.Post("/jobs/{id}/stage", s.handleUpdateStage)
		r.Post("/jobs/{id}/close", s.handleCloseJob)
		r.Get("/jobs/{id}/artifacts/{artifactId}", s.handleArtifactPreview)
	})

	// UI pages
	r.Get(routeRoot, s.handleIndex)
	r.Get("/profile", s.handleProfilePage)
	r.Post("/profile", s.handleProfileFormPost)

	return r
}

func (s *Server) profileRoutes(r chi.Router) {
	r.Get(routeRoot, s.handleGetProfile)
	r.Put(routeRoot, s.handlePutProfile)
}

func (s *Server) jobRoutes(r chi.Router) {
	r.Get(routeRoot, s.handleListJobs)
	r.Post(routeRoot, s.handleCreateJob)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", s.handleGetJob)
		r.Patch("/", s.handleUpdateJob)
		r.Delete("/", s.handleDeleteJob)
		r.Post("/activity", s.handleCreateActivity)
		r.Get("/artifacts", s.handleListArtifacts)
		r.Post("/artifacts", s.handleCreateArtifact)
	})
}

func (s *Server) handleBoardPanel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	jobs, err := s.store.ListJobs(ctx)
	if err != nil {
		http.Error(w, "failed to load jobs", http.StatusInternalServerError)

		return
	}

	staleJobs := s.store.StaleJobSet(ctx, jobs)

	byStage := make(map[string][]db.Job)
	for _, job := range jobs {
		byStage[job.Stage] = append(byStage[job.Stage], job)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	renderErr := ui.Board(byStage, staleJobs).Render(ctx, w)
	if renderErr != nil {
		log.Printf("render board panel: %v", renderErr)
	}
}

func (s *Server) handleJobDetailPanel(w http.ResponseWriter, r *http.Request) {
	s.jobPanel.HandleDetail(w, r)
}

func (s *Server) handleUpdateStage(w http.ResponseWriter, r *http.Request) {
	s.jobPanel.HandleUpdateStage(w, r)
}

func (s *Server) handleCloseJob(w http.ResponseWriter, r *http.Request) {
	s.jobPanel.HandleCloseJob(w, r)
}

func (s *Server) handleArtifactPreview(w http.ResponseWriter, r *http.Request) {
	s.artifactPanel.HandlePreview(w, r)
}
