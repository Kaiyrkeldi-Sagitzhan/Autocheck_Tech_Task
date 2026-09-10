// Package repository contains all SQLite data access: upserts, queries,
// and import run bookkeeping.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"awesomeProject5/internal/domain"
)

// dbHandle is an interface that both *sql.DB and *sql.Tx implement,
// allowing the Repository to work with transactions.
type dbHandle interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Repository provides persistence methods for cars and import runs.
type Repository struct {
	db    dbHandle
	rawDB *sql.DB // kept for BeginTx
}

// New creates a Repository backed by the given database handle.
func New(db *sql.DB) *Repository {
	return &Repository{db: db, rawDB: db}
}

// BeginTx starts a new database transaction.
func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.rawDB.BeginTx(ctx, nil)
}

// WithTx returns a new Repository that uses the given transaction.
// This allows all repository methods to participate in the same transaction.
func (r *Repository) WithTx(tx *sql.Tx) *Repository {
	return &Repository{db: tx, rawDB: r.rawDB}
}

// UpsertCar inserts a new car or updates an existing one by VIN.
// Returns (created, changed, err):
//   - created=true  → new row inserted
//   - created=false, changed=true  → existing row updated with new values
//   - created=false, changed=false → existing row matched but no values changed
func (r *Repository) UpsertCar(ctx context.Context, car *domain.Car) (bool, bool, error) {
	defectsJSON, err := json.Marshal(car.Defects)
	if err != nil {
		return false, false, err
	}

	now := time.Now().UTC()
	if car.ImportedAt.IsZero() {
		car.ImportedAt = now
	}
	car.UpdatedAt = now

	// Try INSERT first. If the VIN already exists, SQLite returns a constraint
	// violation, and we fall back to UPDATE. This avoids the TOCTOU race
	// condition between SELECT and INSERT.
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO cars (vin, brand, model, year, mileage_km, price, currency, color,
			engine, transmission, body_type, defects, defects_raw, status, source_file, imported_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		car.VIN, car.Brand, car.Model, car.Year, car.MileageKm, car.Price, car.Currency, car.Color,
		car.Engine, car.Transmission, car.BodyType, string(defectsJSON), car.DefectsRaw, car.Status,
		car.SourceFile, car.ImportedAt, car.UpdatedAt,
	)
	if err == nil {
		// New row inserted.
		return true, true, nil
	}

	// Check if the error is a UNIQUE constraint violation.
	// SQLite error code for constraint violation is 19 (SQLITE_CONSTRAINT).
	if !isConstraintError(err) {
		return false, false, err
	}

	// VIN already exists — fetch existing car to detect unchanged rows.
	existing, err := r.CarByVIN(ctx, car.VIN)
	if err != nil {
		return false, false, err
	}

	// Compare all business fields. If identical, skip the update.
	changed := !carsEqual(existing, car)
	if !changed {
		return false, false, nil
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE cars SET brand = ?, model = ?, year = ?, mileage_km = ?, price = ?, currency = ?,
			color = ?, engine = ?, transmission = ?, body_type = ?, defects = ?, defects_raw = ?,
			status = ?, source_file = ?, updated_at = ?
			WHERE vin = ?`,
		car.Brand, car.Model, car.Year, car.MileageKm, car.Price, car.Currency, car.Color,
		car.Engine, car.Transmission, car.BodyType, string(defectsJSON), car.DefectsRaw, car.Status,
		car.SourceFile, car.UpdatedAt, car.VIN,
	)
	return false, true, err
}

// isConstraintError checks if the error is a SQLite constraint violation.
func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite constraint violation has error code 19.
	// The modernc.org/sqlite driver returns errors with specific types.
	// We check the error string as a fallback.
	return strings.Contains(err.Error(), "UNIQUE constraint") ||
		strings.Contains(err.Error(), "constraint")
}

// carsEqual returns true if two cars have identical business data (excluding
// ID, ImportedAt, and UpdatedAt which are managed by the system).
func carsEqual(a, b *domain.Car) bool {
	if a.VIN != b.VIN ||
		a.Brand != b.Brand ||
		a.Model != b.Model ||
		a.Currency != b.Currency ||
		a.Color != b.Color ||
		a.Engine != b.Engine ||
		a.Transmission != b.Transmission ||
		a.BodyType != b.BodyType ||
		a.DefectsRaw != b.DefectsRaw ||
		a.Status != b.Status {
		return false
	}
	// Use helper for *int comparison.
	if !ptrIntEqual(a.Year, b.Year) {
		return false
	}
	if !ptrIntEqual(a.MileageKm, b.MileageKm) {
		return false
	}
	if !ptrIntEqual(a.Price, b.Price) {
		return false
	}
	// Compare defects slices.
	if len(a.Defects) != len(b.Defects) {
		return false
	}
	for i := range a.Defects {
		if a.Defects[i] != b.Defects[i] {
			return false
		}
	}
	return true
}

func ptrIntEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// CreateImportRun creates a new import run record.
func (r *Repository) CreateImportRun(ctx context.Context, run *domain.ImportRun) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO import_runs (trigger_type, triggered_by, file_name, status, rows_total, started_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		run.TriggerType, run.TriggeredBy, run.FileName, run.Status, run.RowsTotal, run.StartedAt,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	run.ID = id
	return nil
}

// RecordImportError inserts a row-level import error.
func (r *Repository) RecordImportError(ctx context.Context, runID int64, rowNumber int, reason, rawRow string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO import_errors (run_id, row_number, reason, raw_row, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		runID, rowNumber, reason, rawRow, time.Now().UTC(),
	)
	return err
}

// FinishImportRun updates the import run with final status and counts.
func (r *Repository) FinishImportRun(ctx context.Context, runID int64, status string, run *domain.ImportRun) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE import_runs SET status = ?, rows_total = ?, created = ?, updated = ?, skipped = ?,
		 error_message = ?, finished_at = ?
		 WHERE id = ?`,
		status, run.RowsTotal, run.Created, run.Updated, run.Skipped, run.ErrorMessage, run.FinishedAt, runID,
	)
	return err
}

// ListCars returns a paginated list of cars and the total count.
func (r *Repository) ListCars(ctx context.Context, page, limit int) ([]domain.Car, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cars").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, vin, brand, model, year, mileage_km, price, currency, color,
			engine, transmission, body_type, defects, defects_raw, status, source_file, imported_at, updated_at
		 FROM cars ORDER BY id DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var cars []domain.Car
	for rows.Next() {
		var car domain.Car
		var defectsJSON string
		var importedAt, updatedAt string
		err := rows.Scan(
			&car.ID, &car.VIN, &car.Brand, &car.Model, &car.Year, &car.MileageKm, &car.Price,
			&car.Currency, &car.Color, &car.Engine, &car.Transmission, &car.BodyType,
			&defectsJSON, &car.DefectsRaw, &car.Status, &car.SourceFile, &importedAt, &updatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if defectsJSON != "" {
			if err := json.Unmarshal([]byte(defectsJSON), &car.Defects); err != nil {
				car.Defects = nil
			}
		}

		if t, err := time.Parse(time.RFC3339, importedAt); err == nil {
			car.ImportedAt = t
		}
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			car.UpdatedAt = t
		}

		cars = append(cars, car)
	}

	return cars, total, nil
}

// CarByVIN retrieves a car by VIN for test verification.
func (r *Repository) CarByVIN(ctx context.Context, vin string) (*domain.Car, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, vin, brand, model, year, mileage_km, price, currency, color,
			engine, transmission, body_type, defects, defects_raw, status, source_file, imported_at, updated_at
		 FROM cars WHERE vin = ?`, vin,
	)

	var car domain.Car
	var defectsJSON string
	var importedAt, updatedAt string
	err := row.Scan(
		&car.ID, &car.VIN, &car.Brand, &car.Model, &car.Year, &car.MileageKm, &car.Price,
		&car.Currency, &car.Color, &car.Engine, &car.Transmission, &car.BodyType,
		&defectsJSON, &car.DefectsRaw, &car.Status, &car.SourceFile, &importedAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if defectsJSON != "" {
		if err := json.Unmarshal([]byte(defectsJSON), &car.Defects); err != nil {
			car.Defects = nil
		}
	}

	// Parse timestamps
	if t, err := time.Parse(time.RFC3339, importedAt); err == nil {
		car.ImportedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		car.UpdatedAt = t
	}

	return &car, nil
}

// CountCars returns the total number of cars in the database.
func (r *Repository) CountCars(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cars").Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// LastImportRun returns the most recent import run, or nil if none exist.
func (r *Repository) LastImportRun(ctx context.Context) (*domain.ImportRun, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, trigger_type, triggered_by, file_name, status, rows_total, created, updated, skipped,
			error_message, started_at, finished_at
		 FROM import_runs ORDER BY id DESC LIMIT 1`,
	)

	var run domain.ImportRun
	var finishedAt sql.NullString
	err := row.Scan(
		&run.ID, &run.TriggerType, &run.TriggeredBy, &run.FileName, &run.Status,
		&run.RowsTotal, &run.Created, &run.Updated, &run.Skipped, &run.ErrorMessage,
		&run.StartedAt, &finishedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if finishedAt.Valid {
		t, err := time.Parse(time.RFC3339, finishedAt.String)
		if err == nil {
			run.FinishedAt = &t
		}
	}
	return &run, nil
}
