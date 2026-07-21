package api_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleParse_InvalidBody(t *testing.T) {
	ts, _ := newServer(t)

	resp, err := http.Post(ts.URL+"/api/parse", "application/json", bytes.NewReader([]byte(`not-json`))) //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleParse_MissingURL(t *testing.T) {
	ts, _ := newServer(t)

	resp, err := http.Post(ts.URL+"/api/parse", "application/json", bytes.NewReader([]byte(`{}`))) //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleParse_InvalidURL(t *testing.T) {
	ts, _ := newServer(t)

	// A URL that doesn't match any ATS and fails to fetch (not a real server)
	body := bytes.NewReader([]byte(`{"url":"https://this-host-does-not-exist.invalid/jobs/123"}`))

	resp, err := http.Post(ts.URL+"/api/parse", "application/json", body) //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}
