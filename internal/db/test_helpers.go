package db

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
	"github.com/sdroscher/job-search-pipeline/internal/migrate"
)

// NewTestStore returns an in-memory Store for use in tests.
func NewTestStore(t *testing.T) *Store {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })

	err = migrate.Run(sqlDB)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return &Store{db: sqlDB, Queries: New(sqlDB)}
}
