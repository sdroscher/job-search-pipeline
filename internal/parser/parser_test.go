package parser_test

import (
	"context"
	"testing"

	"github.com/sdroscher/job-search-pipeline/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Dispatch(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"greenhouse", "https://boards.greenhouse.io/co/jobs/1"},
		{"ashby", "https://jobs.ashbyhq.com/co/abc-123"},
		{"lever", "https://jobs.lever.co/co/abc-123"},
		{"html", "https://no-such-host.invalid/jobs/swe"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parser.Parse(context.Background(), tc.url)
			// Should error (no real server reachable), but NOT with a dispatch-level
			// error such as "unknown". The error proves it got past ATS detection and
			// attempted an HTTP call to the appropriate backend.
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "unknown")
		})
	}
}
