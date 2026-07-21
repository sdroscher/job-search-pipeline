package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfile_GetEmpty(t *testing.T) {
	ts, _ := newServer(t)

	resp, err := http.Get(ts.URL + "/api/profile") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	// Returns 404 when no profile exists yet.
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestProfile_PutAndGet(t *testing.T) {
	ts, _ := newServer(t)

	body, err := json.Marshal(map[string]any{
		"resume_md":   "# Simon Droscher\n\nStaff SWE with 20 years experience.",
		"salary_min":  180000,
		"remote_pref": "remote-only",
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, ts.URL+"/api/profile", bytes.NewReader(body)) //nolint:noctx
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp2, err := http.Get(ts.URL + "/api/profile") //nolint:noctx
	require.NoError(t, err)

	defer resp2.Body.Close()

	var profile map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&profile))

	assert.Equal(t, "# Simon Droscher\n\nStaff SWE with 20 years experience.", profile["resume_md"])
	assert.NotEmpty(t, profile["profile_hash"])
}
