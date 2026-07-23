package db_test

import (
	"context"
	"errors"
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

const (
	testYear = 2026
	testDay  = 20
)

func (s *StoreSuite) TestCreateAndListJobs() {
	ctx := context.Background()
	now := time.Date(testYear, time.July, testDay, 0, 0, 0, 0, time.UTC)

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

func (s *StoreSuite) TestWithTx_Commit() {
	ctx := context.Background()
	now := time.Date(testYear, time.July, testDay, 0, 0, 0, 0, time.UTC)

	err := s.store.WithTx(ctx, func(q *db.Queries) error {
		_, txErr := q.CreateJob(ctx, db.CreateJobParams{
			ID: "tx-job", Company: "TxCo", Role: "Dev", Stage: "Evaluated", Verdict: "green",
			Added: now, LastActivity: now,
		})

		return txErr
	})
	s.Require().NoError(err)

	jobs, err := s.store.ListJobs(ctx)
	s.Require().NoError(err)
	s.Require().Len(jobs, 1)
	s.Equal("tx-job", jobs[0].ID)
}

func (s *StoreSuite) TestWithTx_Rollback() {
	ctx := context.Background()

	err := s.store.WithTx(ctx, func(_ *db.Queries) error {
		return errors.New("intentional failure")
	})
	s.Require().Error(err)

	jobs, err := s.store.ListJobs(ctx)
	s.Require().NoError(err)
	s.Empty(jobs, "rollback should leave no jobs")
}
