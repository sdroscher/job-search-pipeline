package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/sdroscher/job-search-pipeline/internal/db"
)

type profileRequest struct {
	ResumeMd          string  `json:"resume_md"`
	CoverLetterSample *string `json:"cover_letter_sample"`
	SalaryMin         *int64  `json:"salary_min"`
	SalaryMax         *int64  `json:"salary_max"`
	SalaryTarget      *int64  `json:"salary_target"`
	RemotePref        *string `json:"remote_pref"`
	Location          *string `json:"location"`
	Industries        *string `json:"industries"`
	GreenFlags        *string `json:"green_flags"`
	RedFlags          *string `json:"red_flags"`
	TechPrefs         *string `json:"tech_prefs"`
	WritingVoiceMd    *string `json:"writing_voice_md"`
	AchievementsMd    *string `json:"achievements_md"`
	CareerNotesMd     *string `json:"career_notes_md"`

	// Server-owned fields, accepted and ignored. PUT replaces the whole
	// profile, so changing one field means GET, edit, PUT the rest back — and
	// a GET response carries these three. Rejecting them would fail the
	// round trip that the merge in /job-search init step 5 depends on.
	// profile_hash is always recomputed from resume_md below.
	IgnoredID          json.RawMessage `json:"id"`
	IgnoredProfileHash json.RawMessage `json:"profile_hash"`
	IgnoredUpdatedAt   json.RawMessage `json:"updated_at"`
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := s.store.GetProfile(r.Context())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "no profile", http.StatusNotFound)

			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handlePutProfile(w http.ResponseWriter, r *http.Request) {
	var req profileRequest

	err := readJSON(r, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(req.ResumeMd)))

	profile, err := s.store.UpsertProfile(r.Context(), db.UpsertProfileParams{
		ResumeMd:          req.ResumeMd,
		CoverLetterSample: req.CoverLetterSample,
		SalaryMin:         req.SalaryMin,
		SalaryMax:         req.SalaryMax,
		SalaryTarget:      req.SalaryTarget,
		RemotePref:        req.RemotePref,
		Location:          req.Location,
		Industries:        req.Industries,
		GreenFlags:        req.GreenFlags,
		RedFlags:          req.RedFlags,
		TechPrefs:         req.TechPrefs,
		WritingVoiceMd:    req.WritingVoiceMd,
		AchievementsMd:    req.AchievementsMd,
		CareerNotesMd:     req.CareerNotesMd,
		ProfileHash:       hash,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	stalErr := s.store.MarkArtifactsStale(r.Context())
	if stalErr != nil {
		log.Printf("mark artifacts stale: %v", stalErr)
	}

	writeJSON(w, http.StatusOK, profile)
}
