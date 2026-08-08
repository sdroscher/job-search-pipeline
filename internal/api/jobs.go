package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

// paramID is the job id, taken from the URL path on every /api/jobs/{id} route.
const paramID = "id"

// errMissingFields is returned when a create request omits a required field.
var errMissingFields = errors.New("missing required field(s)")

// writeJSON sets Content-Type, status, and encodes v as JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		// Header already written; log the encoding error.
		log.Printf("writeJSON encode: %v", err)
	}
}

// readJSON decodes the request body into v, limiting reads to 10 MB.
//
// Unknown fields are rejected rather than dropped. Every client here is an LLM
// working from a written schema, and a silently ignored field name reads as a
// successful write: POSTing /api/parse's "title" to /api/jobs used to create a
// job with an empty role and return 201.
func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 10<<20))
	dec.DisallowUnknownFields()

	return dec.Decode(v)
}

// parseDate parses a "YYYY-MM-DD" string; returns today (UTC, midnight) if s is empty.
func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC().Truncate(24 * time.Hour), nil
	}

	return time.Parse("2006-01-02", s)
}

// createJobRequest is the JSON body accepted by handleCreateJob.
// added and last_activity are optional YYYY-MM-DD strings; both default to today.
type createJobRequest struct {
	ID            string  `json:"id"`
	Company       string  `json:"company"`
	Role          string  `json:"role"`
	Stage         string  `json:"stage"`
	Verdict       string  `json:"verdict"`
	Salary        *string `json:"salary"`
	SalaryMin     *int64  `json:"salary_min"`
	Remote        *string `json:"remote"`
	Source        *string `json:"source"`
	SourceURL     *string `json:"source_url"`
	RawJd         *string `json:"raw_jd"`
	Added         string  `json:"added"`         // "YYYY-MM-DD" or ""
	LastActivity  string  `json:"last_activity"` // "YYYY-MM-DD" or ""
	FitScore      *int64  `json:"fit_score"`
	Summary       *string `json:"summary"`
	Positives     *string `json:"positives"`
	Concerns      *string `json:"concerns"`
	MyNotes       *string `json:"my_notes"`
	CompanyValues *string `json:"company_values"`
	Networking    *string `json:"networking"`
	RoleDetails   *string `json:"role_details"`
}

// missingRequired names the fields a job cannot be created without. Without
// this, a typo in a field name produces a job with an empty role or company.
func (req createJobRequest) missingRequired() []string {
	var missing []string

	if req.ID == "" {
		missing = append(missing, paramID)
	}

	if req.Company == "" {
		missing = append(missing, "company")
	}

	if req.Role == "" {
		missing = append(missing, "role")
	}

	return missing
}

// toParams validates the request and maps it onto CreateJobParams. Errors are
// safe to return to the client verbatim.
func (req createJobRequest) toParams() (db.CreateJobParams, error) {
	missing := req.missingRequired()
	if len(missing) > 0 {
		return db.CreateJobParams{}, fmt.Errorf("%w: %s", errMissingFields, strings.Join(missing, ", "))
	}

	added, err := parseDate(req.Added)
	if err != nil {
		return db.CreateJobParams{}, fmt.Errorf("invalid added date: %w", err)
	}

	lastActivity, err := parseDate(req.LastActivity)
	if err != nil {
		return db.CreateJobParams{}, fmt.Errorf("invalid last_activity date: %w", err)
	}

	return db.CreateJobParams{
		ID:            req.ID,
		Company:       req.Company,
		Role:          req.Role,
		Stage:         req.Stage,
		Verdict:       req.Verdict,
		Salary:        req.Salary,
		SalaryMin:     req.SalaryMin,
		Remote:        req.Remote,
		Source:        req.Source,
		SourceUrl:     req.SourceURL,
		RawJd:         req.RawJd,
		Added:         added,
		LastActivity:  lastActivity,
		FitScore:      req.FitScore,
		Summary:       req.Summary,
		Positives:     req.Positives,
		Concerns:      req.Concerns,
		MyNotes:       req.MyNotes,
		CompanyValues: req.CompanyValues,
		Networking:    req.Networking,
		RoleDetails:   req.RoleDetails,
	}, nil
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.store.ListJobs(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// Return [] not null when empty.
	if jobs == nil {
		jobs = []db.Job{}
	}

	writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest

	err := readJSON(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	params, err := req.toParams()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	job, err := s.store.CreateJob(r.Context(), params)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			http.Error(w, "job with this ID already exists", http.StatusConflict)

			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	_, actErr := s.store.CreateActivityEntry(r.Context(), db.CreateActivityEntryParams{
		JobID:  job.ID,
		Date:   time.Now().UTC().Truncate(24 * time.Hour),
		Action: "Added",
	})
	if actErr != nil {
		log.Printf("create activity entry for %s: %v", job.ID, actErr)
	}

	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, paramID)

	job, err := s.store.GetJob(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleUpdateJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, paramID)

	var params db.UpdateJobParams

	err := readJSON(r, &params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	params.ID = id

	job, err := s.store.UpdateJob(r.Context(), params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, paramID)

	_, getErr := s.store.GetJob(r.Context(), id)
	if getErr != nil {
		if errors.Is(getErr, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, getErr.Error(), http.StatusInternalServerError)
		}

		return
	}

	delErr := s.store.DeleteJob(r.Context(), id)
	if delErr != nil {
		http.Error(w, delErr.Error(), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
