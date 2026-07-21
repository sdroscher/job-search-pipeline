package feature_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ts, _ := newServerWithStore(t)

	return ts
}

func TestBoard_EmptyRendersAllColumns(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/") //nolint:noctx
	require.NoError(t, err)

	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	body := string(raw)

	for _, stage := range []string{"Evaluated", "Applied", "AI Assessment", "Screening", "Interviewing", "Final Round", "Offer"} {
		assert.Contains(t, body, stage, "board missing column %q", stage)
	}
}
