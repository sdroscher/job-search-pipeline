package migrate_test

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
	"github.com/sdroscher/job-search-pipeline/internal/migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_CreatesSchema(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	defer db.Close()

	err = migrate.Run(db)
	require.NoError(t, err, "migrate.Run")

	ctx := context.Background()

	// verify tables exist
	for _, table := range []string{"user_profile", "jobs", "activity_log", "artifacts"} {
		var name string

		tableErr := db.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		require.NoError(t, tableErr, "table %q not found", table)
	}

	// verify added columns exist in user_profile
	for _, col := range []string{"achievements_md", "career_notes_md"} {
		var count int

		colErr := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM pragma_table_info('user_profile') WHERE name=?", col,
		).Scan(&count)
		require.NoError(t, colErr)
		assert.Equal(t, 1, count, "column %q not found in user_profile", col)
	}
}

// TestRun_AddsColumnToExistingDB verifies that Run upgrades a database that was
// created before achievements_md existed (simulates a user upgrading from an older version).
func TestRun_AddsColumnToExistingDB(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	defer db.Close()

	ctx := context.Background()

	// Create user_profile without achievements_md, simulating a pre-migration database.
	_, err = db.ExecContext(ctx, `CREATE TABLE user_profile (
		id INTEGER PRIMARY KEY DEFAULT 1,
		resume_md TEXT NOT NULL DEFAULT '',
		writing_voice_md TEXT,
		profile_hash TEXT NOT NULL DEFAULT '',
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	err = migrate.Run(db)
	require.NoError(t, err, "migrate.Run on existing db")

	for _, col := range []string{"achievements_md", "career_notes_md"} {
		var count int

		colErr := db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM pragma_table_info('user_profile') WHERE name=?", col,
		).Scan(&count)
		require.NoError(t, colErr)
		assert.Equal(t, 1, count, "column %q should be added to existing table", col)
	}
}
