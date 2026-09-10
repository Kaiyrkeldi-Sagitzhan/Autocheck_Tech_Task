// Package domain contains the core business entities shared across layers.
package domain

import "time"

// CarStatus represents the lifecycle state of a car in inventory.
type CarStatus string

const (
	CarStatusActive   CarStatus = "active"
	CarStatusSold     CarStatus = "sold"
	CarStatusReserved CarStatus = "reserved"
)

// Car is a normalized inventory record, uniquely identified by VIN.
type Car struct {
	ID           int64     `json:"id"`
	VIN          string    `json:"vin"`
	Brand        string    `json:"brand"`
	Model        string    `json:"model"`
	Year         *int      `json:"year,omitempty"`
	MileageKm    *int      `json:"mileage_km,omitempty"`
	Price        *int      `json:"price,omitempty"`
	Currency     string    `json:"currency"`
	Color        string    `json:"color,omitempty"`
	Engine       string    `json:"engine,omitempty"`
	Transmission string    `json:"transmission,omitempty"`
	BodyType     string    `json:"body_type,omitempty"`
	Defects      []string  `json:"defects,omitempty"`
	DefectsRaw   string    `json:"defects_raw,omitempty"`
	Status       CarStatus `json:"status"`
	SourceFile   string    `json:"source_file,omitempty"`
	ImportedAt   time.Time `json:"imported_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CarRecord is a single parsed row from a 1C export file, before persistence.
type CarRecord struct {
	VIN          string
	Brand        string
	Model        string
	Year         *int
	MileageKm    *int
	Price        *int
	Currency     string
	Color        string
	Engine       string
	Transmission string
	BodyType     string
	DefectsRaw   string
	Status       CarStatus
	UpdatedAt    string
	RowNumber    int
}

// ImportRun tracks one execution of the import process.
type ImportRun struct {
	ID           int64
	TriggerType  string // "scheduled" | "manual"
	TriggeredBy  string // e.g. "cron", "api"
	FileName     string
	Status       string // "running" | "success" | "failed"
	RowsTotal    int
	Created      int
	Updated      int
	Skipped      int
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   *time.Time
}

// ImportError is a single row that failed during an import run.
type ImportError struct {
	ID        int64
	RunID     int64
	RowNumber int
	Reason    string
	RawRow    string
	CreatedAt time.Time
}

// ImportReport is the summary returned after an import completes.
type ImportReport struct {
	File      string         `json:"file"`
	RowsTotal int            `json:"rows_total"`
	Created   int            `json:"created"`
	Updated   int            `json:"updated"`
	Skipped   int            `json:"skipped"`
	Errors    []RowErrorInfo `json:"errors"`
}

// RowErrorInfo describes one failed row in an import report.
type RowErrorInfo struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}
