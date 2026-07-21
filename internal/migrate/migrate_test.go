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

		err := db.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		assert.NoError(t, err, "table %q not found", table)
	}
}
