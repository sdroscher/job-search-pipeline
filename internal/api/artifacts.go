package api

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

	_, err := s.store.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "job not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

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

	_, jobErr := s.store.GetJob(r.Context(), id)
	if jobErr != nil {
		if errors.Is(jobErr, sql.ErrNoRows) {
			http.Error(w, "job not found", http.StatusNotFound)
		} else {
			http.Error(w, jobErr.Error(), http.StatusInternalServerError)
		}

		return
	}

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

	// Validate that the requested filepath is contained within the configured output directory.
	cleanPath := filepath.Clean(req.Filepath)
	outputDir := filepath.Clean(s.config.OutputDir)

	if !strings.HasPrefix(cleanPath, outputDir+string(filepath.Separator)) && cleanPath != outputDir {
		http.Error(w, "filepath must be under output directory", http.StatusBadRequest)

		return
	}

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
