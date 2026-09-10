// Package repository contains all SQLite data access: upserts, queries,
// and import run bookkeeping.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"awesomeProject5/internal/domain"
)

// Repository provides persistence methods for cars and import runs.
type Repository struct {
	db *sql.DB
}

// New creates a Repository backed by the given database handle.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// UpsertCar inserts a new car or updates an existing one by VIN.
// Returns (true, nil) if a new row was inserted, (false, nil) if an existing row was updated.
func (r *Repository) UpsertCar(ctx context.Context, car *domain.Car) (bool, error) {
	// Check if VIN already exists to determine insert vs update.
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM cars WHERE vin = ?)", car.VIN,
	).Scan(&exists)
	if err != nil {
		return false, err
	}

	defectsJSON, err := json.Marshal(car.Defects)
	if err != nil {
		return false, err
	}

	now := time.Now().UTC()
	if car.ImportedAt.IsZero() {
		car.ImportedAt = now
	}
	car.UpdatedAt = now

	if !exists {
		_, err = r.db.ExecContext(ctx,
			`INSERT INTO cars (vin, brand, model, year, mileage_km, price, currency, color,
				engine, transmission, body_type, defects, defects_raw, status, source_file, imported_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			car.VIN, car.Brand, car.Model, car.Year, car.MileageKm, car.Price, car.Currency, car.Color,
			car.Engine, car.Transmission, car.BodyType, string(defectsJSON), car.DefectsRaw, car.Status,
			car.SourceFile, car.ImportedAt, car.UpdatedAt,
		)
		return true, err
	}

	// Update existing car.
	_, err = r.db.ExecContext(ctx,
		`UPDATE cars SET brand = ?, model = ?, year = ?, mileage_km = ?, price = ?, currency = ?,
			color = ?, engine = ?, transmission = ?, body_type = ?, defects = ?, defects_raw = ?,
			status = ?, source_file = ?, updated_at = ?
			WHERE vin = ?`,
		car.Brand, car.Model, car.Year, car.MileageKm, car.Price, car.Currency, car.Color,
		car.Engine, car.Transmission, car.BodyType, string(defectsJSON), car.DefectsRaw, car.Status,
		car.SourceFile, car.UpdatedAt, car.VIN,
	)
	return false, err
}

// CreateImportRun creates a new import run record.
func (r *Repository) CreateImportRun(ctx context.Context, run *domain.ImportRun) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO import_runs (trigger_type, file_name, status, rows_total, started_at)
		 VALUES (?, ?, ?, ?, ?)`,
		run.TriggerType, run.FileName, run.Status, run.RowsTotal, run.StartedAt,
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
