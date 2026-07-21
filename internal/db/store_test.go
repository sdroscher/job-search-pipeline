package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/sdroscher/job-search-pipeline/internal/db"
)

func TestStore_UpsertAndGetProfile(t *testing.T) {
	s := db.NewTestStore(t)
	ctx := context.Background()

	_, err := s.UpsertProfile(ctx, db.UpsertProfileParams{
		ResumeMd:    "# My Resume",
		ProfileHash: "abc123",
	})
	if err != nil {
		t.Fatalf("UpsertProfile: %v", err)
	}

	got, err := s.GetProfile(ctx)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if got.ResumeMd != "# My Resume" {
		t.Errorf("got resume %q, want %q", got.ResumeMd, "# My Resume")
	}
}

func TestStore_CreateAndListJobs(t *testing.T) {
	s := db.NewTestStore(t)
	ctx := context.Background()

	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	_, err := s.CreateJob(ctx, db.CreateJobParams{
		ID:           "acme-staff-swe",
		Company:      "Acme",
		Role:         "Staff SWE",
		Stage:        "Evaluated",
		Verdict:      "green",
		Added:        now,
		LastActivity: now,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	jobs, err := s.ListJobs(ctx)
	if err != nil {
		t.Fatalf("ListJobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("got %d jobs, want 1", len(jobs))
	}
	if jobs[0].Company != "Acme" {
		t.Errorf("got company %q, want %q", jobs[0].Company, "Acme")
	}
}
