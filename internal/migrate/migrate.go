package migrate

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schema string

// Run applies the schema and any additive column migrations to db.
// Safe to call on both fresh and existing databases.
func Run(db *sql.DB) error {
	ctx := context.Background()

	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	// Column additions for existing databases — add new calls here as columns are added.
	err = addColumnIfMissing(ctx, db, "user_profile", "achievements_md", "TEXT")
	if err != nil {
		return err
	}

	err = addColumnIfMissing(ctx, db, "user_profile", "career_notes_md", "TEXT")
	if err != nil {
		return err
	}

	return nil
}

// addColumnIfMissing adds column to table if it does not already exist.
// SQLite does not support ALTER TABLE … ADD COLUMN IF NOT EXISTS, so we
// inspect PRAGMA table_info first.
func addColumnIfMissing(ctx context.Context, db *sql.DB, table, column, colType string) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return fmt.Errorf("table_info %s: %w", table, err)
	}

	defer rows.Close()

	for rows.Next() {
		var cid, notNull, pk int
		var name, typeName string
		var defaultVal sql.NullString

		scanErr := rows.Scan(&cid, &name, &typeName, &notNull, &defaultVal, &pk)
		if scanErr != nil {
			return fmt.Errorf("scan table_info %s: %w", table, scanErr)
		}

		if name == column {
			return nil
		}
	}

	rowsErr := rows.Err()
	if rowsErr != nil {
		return fmt.Errorf("table_info %s: %w", table, rowsErr)
	}

	_, err = db.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN "+column+" "+colType)
	if err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}

	return nil
}
