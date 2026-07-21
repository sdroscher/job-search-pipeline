package migrate

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schema string

func Run(db *sql.DB) error {
	_, err := db.ExecContext(context.Background(), schema)
	if err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	return nil
}
