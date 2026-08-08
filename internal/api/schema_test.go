package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validJobBody is a minimal create request with every required field set.
const validJobBody = `{"id":"acme-swe","company":"Acme","role":"Staff Engineer"}`

// postJSON POSTs a raw JSON body and returns the status and body text.
func postJSON(t *testing.T, method, target, body string) (int, string) {
	t.Helper()

	req, err := http.NewRequest(method, target, strings.NewReader(body)) //nolint:noctx
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	out, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(out)
}

// A client that copies /api/parse's "title" into /api/jobs must be told it is
// wrong rather than silently creating a job with an empty role.
func TestCreateJob_UnknownFieldRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		`{"id":"acme-swe","company":"Acme","title":"Staff Engineer"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "title")

	// And nothing was created.
	resp, err := http.Get(ts.URL + "/api/jobs/acme-swe") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateJob_EmptyRoleRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		`{"id":"acme-swe","company":"Acme"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "role")
}

func TestCreateJob_EmptyCompanyRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		`{"id":"acme-swe","role":"Staff Engineer"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "company")
}

func TestCreateJob_EmptyIDRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		`{"company":"Acme","role":"Staff Engineer"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "id")
}

func TestUpdateJob_UnknownFieldRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, _ := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		validJobBody)
	require.Equal(t, http.StatusCreated, status)

	status, body := postJSON(t, http.MethodPatch, ts.URL+"/api/jobs/acme-swe",
		`{"title":"Principal Engineer"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "title")
}

// The patched UpdateJob query covers role and remote, so a job created with the
// wrong title field can be repaired via PATCH.
func TestUpdateJob_RoleAndRemote(t *testing.T) {
	ts, _ := newServer(t)

	status, _ := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		validJobBody)
	require.Equal(t, http.StatusCreated, status)

	status, body := postJSON(t, http.MethodPatch, ts.URL+"/api/jobs/acme-swe",
		`{"role":"Principal Engineer","remote":"remote"}`)
	require.Equal(t, http.StatusOK, status, body)

	var job map[string]any
	require.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(&job))

	assert.Equal(t, "Principal Engineer", job["role"])
	assert.Equal(t, "remote", job["remote"])
	assert.Equal(t, "Acme", job["company"], "unset fields must be preserved")
}

// Every column a client can get wrong on create must be fixable in place.
// DeleteJob is a soft delete that keeps the row, so "delete and recreate" is
// not an available remedy: the id stays taken and the recreate 409s.
func TestUpdateJob_AllCorrectableFields(t *testing.T) {
	ts, _ := newServer(t)

	status, _ := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		`{"id":"acme-swe","company":"Acme Crop","role":"Staff Engineer","source":"LinkedIn"}`)
	require.Equal(t, http.StatusCreated, status)

	status, body := postJSON(t, http.MethodPatch, ts.URL+"/api/jobs/acme-swe",
		`{"company":"Acme","source":"Greenhouse","source_url":"https://boards.greenhouse.io/acme/1","raw_jd":"# Staff Engineer"}`)
	require.Equal(t, http.StatusOK, status, body)

	var job map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &job))

	assert.Equal(t, "Acme", job["company"])
	assert.Equal(t, "Greenhouse", job["source"])
	assert.Equal(t, "https://boards.greenhouse.io/acme/1", job["source_url"])
	assert.Equal(t, "# Staff Engineer", job["raw_jd"])
	assert.Equal(t, "Staff Engineer", job["role"], "unset fields must be preserved")
}

// Pins the soft-delete semantics the PATCH docs now warn about.
func TestDeleteJob_KeepsIDTaken(t *testing.T) {
	ts, _ := newServer(t)

	status, _ := postJSON(t, http.MethodPost, ts.URL+"/api/jobs", validJobBody)
	require.Equal(t, http.StatusCreated, status)

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/api/jobs/acme-swe", nil) //nolint:noctx
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/jobs", validJobBody)
	assert.Equal(t, http.StatusConflict, status)
	assert.Contains(t, body, "already exists")
}

func TestPutProfile_UnknownFieldRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, body := postJSON(t, http.MethodPut, ts.URL+"/api/profile",
		`{"resume_md":"# Simon","achievements":"nope"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "achievements")
}

// /job-search init collects career notes and an achievement bank; both must
// survive a round trip or the resume and cover-letter commands read back null.
func TestPutProfile_AchievementsAndCareerNotes(t *testing.T) {
	ts, _ := newServer(t)

	body, err := json.Marshal(map[string]any{
		"resume_md":       "# Simon Droscher",
		"achievements_md": "## Scale\n- Ran a 40k rps ingest pipeline.",
		"career_notes_md": "## Staff SWE\n**Outcome:** cut p99 by 60%.",
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/profile", bytes.NewReader(body)) //nolint:noctx
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp2, err := http.Get(ts.URL + "/api/profile") //nolint:noctx
	require.NoError(t, err)

	defer resp2.Body.Close()

	var profile map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&profile))

	assert.Equal(t, "## Scale\n- Ran a 40k rps ingest pipeline.", profile["achievements_md"])
	assert.Equal(t, "## Staff SWE\n**Outcome:** cut p99 by 60%.", profile["career_notes_md"])
}

// The profile page form has no inputs for these two fields, so saving it must
// not wipe what /job-search init collected.
func TestProfileForm_PreservesAchievements(t *testing.T) {
	ts, _ := newServer(t)

	body, err := json.Marshal(map[string]any{
		"resume_md":       "# Simon Droscher",
		"achievements_md": "## Scale\n- Ran a 40k rps ingest pipeline.",
		"career_notes_md": "## Staff SWE\n**Outcome:** cut p99 by 60%.",
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/profile", bytes.NewReader(body)) //nolint:noctx
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	form := url.Values{}
	form.Set("resume_md", "# Simon Droscher (edited)")
	form.Set("location", "Vancouver, Canada")

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	formResp, err := client.PostForm(ts.URL+"/profile", form) //nolint:noctx
	require.NoError(t, err)

	defer formResp.Body.Close()
	require.Equal(t, http.StatusSeeOther, formResp.StatusCode)

	resp2, err := http.Get(ts.URL + "/api/profile") //nolint:noctx
	require.NoError(t, err)

	defer resp2.Body.Close()

	var profile map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&profile))

	assert.Equal(t, "# Simon Droscher (edited)", profile["resume_md"])
	assert.Equal(t, "## Scale\n- Ran a 40k rps ingest pipeline.", profile["achievements_md"])
	assert.Equal(t, "## Staff SWE\n**Outcome:** cut p99 by 60%.", profile["career_notes_md"])
}

// /job-search init step 5 merges by GETting the profile, editing it, and
// PUTting it back. The GET response carries id, profile_hash and updated_at,
// so that round trip has to succeed.
func TestPutProfile_RoundTripsGetResponse(t *testing.T) {
	ts, _ := newServer(t)

	status, _ := postJSON(t, http.MethodPut, ts.URL+"/api/profile",
		`{"resume_md":"# Simon","achievements_md":"## Scale\n- 40k rps."}`)
	require.Equal(t, http.StatusOK, status)

	resp, err := http.Get(ts.URL + "/api/profile") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	stored, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Send the GET response back verbatim, as a naive merge would.
	status, body := postJSON(t, http.MethodPut, ts.URL+"/api/profile", string(stored))
	require.Equal(t, http.StatusOK, status, body)

	var profile map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &profile))

	assert.Equal(t, "# Simon", profile["resume_md"])
	assert.Equal(t, "## Scale\n- 40k rps.", profile["achievements_md"], "merge must not drop existing fields")

	// A genuinely unknown field is still rejected.
	status, body = postJSON(t, http.MethodPut, ts.URL+"/api/profile",
		`{"resume_md":"# Simon","achievement_bank":"wrong name"}`)
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "achievement_bank")
}

func TestCreateActivity_UnknownFieldRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, _ := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		validJobBody)
	require.Equal(t, http.StatusCreated, status)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/jobs/acme-swe/activity",
		`{"action":"Evaluated","note":"typo, should be notes"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "note")
}

func TestCreateArtifact_UnknownFieldRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, _ := postJSON(t, http.MethodPost, ts.URL+"/api/jobs",
		validJobBody)
	require.Equal(t, http.StatusCreated, status)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/jobs/acme-swe/artifacts",
		`{"type":"resume","path":"/tmp/x.md","profile_hash":"abc"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "path")
}

func TestParse_UnknownFieldRejected(t *testing.T) {
	ts, _ := newServer(t)

	status, body := postJSON(t, http.MethodPost, ts.URL+"/api/parse",
		`{"link":"https://example.com/job"}`)

	assert.Equal(t, http.StatusBadRequest, status)
	assert.Contains(t, body, "link")
}
