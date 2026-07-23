package api

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/sdroscher/job-search-pipeline/internal/ui"
)

func (s *Server) handleProfilePage(w http.ResponseWriter, r *http.Request) {
	profile, err := s.store.GetProfile(r.Context())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	saved := r.URL.Query().Get("saved") == "1"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	renderErr := ui.ProfilePage(profile, saved).Render(r.Context(), w)
	if renderErr != nil {
		log.Printf("render profile page: %v", renderErr)
	}
}

func (s *Server) handleProfileFormPost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)

		return
	}

	resumeMd := r.FormValue("resume_md")
	if resumeMd == "" {
		http.Error(w, "resume_md required", http.StatusBadRequest)

		return
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(resumeMd)))

	_, upsertErr := s.store.UpsertProfile(r.Context(), db.UpsertProfileParams{
		ResumeMd:          resumeMd,
		CoverLetterSample: optString(r.FormValue("cover_letter_sample")),
		SalaryMin:         optInt64(r.FormValue("salary_min")),
		SalaryMax:         optInt64(r.FormValue("salary_max")),
		SalaryTarget:      optInt64(r.FormValue("salary_target")),
		RemotePref:        optString(r.FormValue("remote_pref")),
		Location:          optString(r.FormValue("location")),
		Industries:        optString(r.FormValue("industries")),
		GreenFlags:        optString(r.FormValue("green_flags")),
		RedFlags:          optString(r.FormValue("red_flags")),
		TechPrefs:         optString(r.FormValue("tech_prefs")),
		WritingVoiceMd:    optString(r.FormValue("writing_voice_md")),
		ProfileHash:       hash,
	})
	if upsertErr != nil {
		http.Error(w, upsertErr.Error(), http.StatusInternalServerError)

		return
	}

	staleErr := s.store.MarkArtifactsStale(r.Context())
	if staleErr != nil {
		log.Printf("mark artifacts stale: %v", staleErr)
	}

	http.Redirect(w, r, "/profile?saved=1", http.StatusSeeOther)
}

func optString(val string) *string {
	if val == "" {
		return nil
	}

	return &val
}

const (
	intBase   = 10
	int64Bits = 64
)

func optInt64(val string) *int64 {
	if val == "" {
		return nil
	}

	parsed, err := strconv.ParseInt(val, intBase, int64Bits)
	if err != nil {
		return nil
	}

	return &parsed
}
