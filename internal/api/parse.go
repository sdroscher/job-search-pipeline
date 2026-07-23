package api

import (
	"net/http"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
)

type parseRequest struct {
	URL string `json:"url"`
}

func (*Server) handleParse(w http.ResponseWriter, r *http.Request) {
	var req parseRequest

	err := readJSON(r, &req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)

		return
	}

	if req.URL == "" {
		http.Error(w, "url required", http.StatusBadRequest)

		return
	}

	job, err := parser.Parse(r.Context(), req.URL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)

		return
	}

	writeJSON(w, http.StatusOK, job)
}
