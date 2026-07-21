package migrate_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
	"github.com/sdroscher/job-search-pipeline/internal/migrate"
)

func TestRun_CreatesSchema(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	defer db.Close()

	err = migrate.Run(db)
	if err != nil {
		t.Fatalf("migrate.Run: %v", err)
	}

	ctx := context.Background()

	// verify tables exist
	for _, table := range []string{"user_profile", "jobs", "activity_log", "artifacts"} {
		var name string

		err := db.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}
