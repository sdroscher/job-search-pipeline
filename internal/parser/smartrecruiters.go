package parser

import (
	"context"
	"errors"
	"fmt"
)

var errSmartRecruitersNotImplemented = errors.New("smartrecruiters parser not yet implemented")

// FetchSmartRecruiters is a placeholder — full implementation comes in Task 2.
func FetchSmartRecruiters(_ context.Context, rawURL string) (*ParsedJob, error) {
	return nil, fmt.Errorf("%s: %w", rawURL, errSmartRecruitersNotImplemented)
}
