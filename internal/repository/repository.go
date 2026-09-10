// Package repository contains all SQLite data access: upserts, queries,
// and import run bookkeeping.
package repository

import "database/sql"

// Repository provides persistence methods for cars and import runs.
type Repository struct {
	db *sql.DB
}

// New creates a Repository backed by the given database handle.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}
