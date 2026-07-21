package db

import (
	"context"
	"database/sql"
	"fmt"

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
