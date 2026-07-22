package feature_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoard(t *testing.T) {
	t.Run("empty board shows all 7 stage columns", func(t *testing.T) {
		ts := newTestServer(t)
		body := getBoard(t, ts)

		for _, stage := range []string{
			"Evaluated", "Applied", "AI Assessment", "Screening",
			"Interviewing", "Final Round", "Offer",
		} {
			assert.Contains(t, body, stage, "board missing column %q", stage)
		}
	})

	t.Run("after a job is added it appears in its stage column", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "board-job-1", "Acme Corp", "Staff SWE", "Applied")

		body := getBoard(t, ts)

		assert.Contains(t, body, "Acme Corp")
	})

	t.Run("stale warning appears after profile update", func(t *testing.T) {
		ts := newTestServer(t)
		createJob(t, ts, "stale-board-job", "Acme", "SWE", "Evaluated")

		profile := putProfile(t, ts, "# Resume v1")
		hash, _ := profile["profile_hash"].(string)

		createArtifact(t, ts, "stale-board-job", "resume", "resume-acme-swe.md", hash)

		// Update profile — makes the artifact stale.
		putProfile(t, ts, "# Resume v2 with new content")

		body := getBoard(t, ts)

		assert.Contains(t, body, "⚠", "board should show stale warning for job with outdated artifact")
	})

	t.Run("GET /panels/board returns HTML fragment containing stage columns", func(t *testing.T) {
		ts := newTestServer(t)
		body := getBoardPanel(t, ts)

		for _, stage := range []string{
			"Evaluated", "Applied", "AI Assessment", "Screening",
			"Interviewing", "Final Round", "Offer",
		} {
			assert.Contains(t, body, stage, "board panel missing column %q", stage)
		}
	})
}
