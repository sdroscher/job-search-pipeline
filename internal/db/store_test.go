package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/sdroscher/job-search-pipeline/internal/db"
	"github.com/stretchr/testify/suite"
)

// StoreSuite groups all Store tests that share a common test-store setup.
type StoreSuite struct {
	suite.Suite
	store *db.Store
}

func (s *StoreSuite) SetupTest() {
	s.store = db.NewTestStore(s.T())
}

func TestStoreSuite(t *testing.T) {
	suite.Run(t, new(StoreSuite))
}

func (s *StoreSuite) TestUpsertAndGetProfile() {
	ctx := context.Background()

	_, err := s.store.UpsertProfile(ctx, db.UpsertProfileParams{
		ResumeMd:    "# My Resume",
		ProfileHash: "abc123",
	})
	s.Require().NoError(err, "UpsertProfile")

	got, err := s.store.GetProfile(ctx)
	s.Require().NoError(err, "GetProfile")
	s.Equal("# My Resume", got.ResumeMd)
}

func (s *StoreSuite) TestCreateAndListJobs() {
	ctx := context.Background()
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)

	_, err := s.store.CreateJob(ctx, db.CreateJobParams{
		ID:           "acme-staff-swe",
		Company:      "Acme",
		Role:         "Staff SWE",
		Stage:        "Evaluated",
		Verdict:      "green",
		Added:        now,
		LastActivity: now,
	})
	s.Require().NoError(err, "CreateJob")

	jobs, err := s.store.ListJobs(ctx)
	s.Require().NoError(err, "ListJobs")
	s.Require().Len(jobs, 1)
	s.Equal("Acme", jobs[0].Company)
}
