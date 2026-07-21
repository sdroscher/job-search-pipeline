package api

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

type createArtifactRequest struct {
	Type        string `json:"type"`
	Filepath    string `json:"filepath"`
	ProfileHash string `json:"profile_hash"`
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	artifacts, err := s.store.ListArtifacts(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if artifacts == nil {
		artifacts = []db.Artifact{}
	}

	writeJSON(w, http.StatusOK, artifacts)
}

func (s *Server) handleCreateArtifact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req createArtifactRequest

	err := readJSON(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	if req.Type == "" || req.Filepath == "" || req.ProfileHash == "" {
		http.Error(w, "type, filepath, and profile_hash required", http.StatusBadRequest)

		return
	}

	// req.Filepath is trusted input from the Claude skill (CLI), not from untrusted
	// external users, so path traversal via this field is acceptable.
	mkdirErr := os.MkdirAll(filepath.Dir(req.Filepath), 0o750)
	if mkdirErr != nil {
		http.Error(w, "cannot create output dir", http.StatusInternalServerError)

		return
	}

	artifact, err := s.store.CreateArtifact(r.Context(), db.CreateArtifactParams{
		JobID:       id,
		Type:        req.Type,
		Filepath:    req.Filepath,
		ProfileHash: req.ProfileHash,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusCreated, artifact)
}
