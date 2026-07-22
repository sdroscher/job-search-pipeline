package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
	"github.com/sdroscher/job-search-pipeline/internal/migrate"
)

// Store wraps sqlc Queries and the raw *sql.DB (for transactions).
type Store struct {
	db *sql.DB
	*Queries
}

func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite3", dsn+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	err = migrate.Run(db)
	if err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db, Queries: New(db)}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// StaleJobSet returns a set of job IDs that have at least one stale artifact.
func (s *Store) StaleJobSet(ctx context.Context, jobs []Job) map[string]bool {
	stale := make(map[string]bool, len(jobs))

	for _, job := range jobs {
		artifacts, err := s.ListArtifacts(ctx, job.ID)
		if err != nil {
			log.Printf("list artifacts for stale check (job=%s): %v", job.ID, err)

			continue
		}

		for _, artifact := range artifacts {
			if artifact.Stale == 1 {
				stale[job.ID] = true

				break
			}
		}
	}

	return stale
}

// WithTx runs callback inside a transaction, rolling back on error.
func (s *Store) WithTx(ctx context.Context, callback func(*Queries) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	err = callback(New(tx))
	if err != nil {
		_ = tx.Rollback()

		return err
	}

	return tx.Commit()
}
