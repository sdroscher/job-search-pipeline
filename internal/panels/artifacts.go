package panels

import (
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

// ArtifactPanelHandler handles HTMX panel requests for artifact preview.
type ArtifactPanelHandler struct {
	store *db.Store
}

// NewArtifactPanelHandler creates a new ArtifactPanelHandler.
func NewArtifactPanelHandler(store *db.Store) *ArtifactPanelHandler {
	return &ArtifactPanelHandler{store: store}
}

// HandlePreview reads the artifact file from disk and returns a <pre> HTML fragment.
func (h *ArtifactPanelHandler) HandlePreview(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	rawArtifactID := chi.URLParam(r, "artifactId")

	artifactID, parseErr := strconv.ParseInt(rawArtifactID, 10, 64)
	if parseErr != nil {
		http.Error(w, "invalid artifact id", http.StatusBadRequest)

		return
	}

	artifact, dbErr := h.store.GetArtifact(r.Context(), db.GetArtifactParams{
		ID:    artifactID,
		JobID: jobID,
	})
	if dbErr != nil {
		if errors.Is(dbErr, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, dbErr.Error(), http.StatusInternalServerError)
		}

		return
	}

	content, readErr := os.ReadFile(artifact.Filepath)
	if readErr != nil {
		log.Printf("artifact preview read %q: %v", artifact.Filepath, readErr)
		http.Error(w, "file not readable", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, writeErr := fmt.Fprintf(w, `<pre class="artifact-pre">%s</pre>`, html.EscapeString(string(content)))
	if writeErr != nil {
		log.Printf("artifact preview write: %v", writeErr)
	}
}
