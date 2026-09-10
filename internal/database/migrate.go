package database

import (
	"database/sql"
	"fmt"
)

// Migrate applies the schema in migrations/001_schema.sql (embedded).
func Migrate(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
