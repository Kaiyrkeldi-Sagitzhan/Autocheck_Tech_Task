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
	ID           int64
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
	Defects      []string // normalized defect codes/strings
	DefectsRaw   string   // original defect field from the 1C export
	Status       CarStatus
	SourceFile   string
	ImportedAt   time.Time
	UpdatedAt    time.Time
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
	RowNumber    int
}

// ImportRun tracks one execution of the import process.
type ImportRun struct {
	ID           int64
	TriggerType  string // "scheduled" | "manual"
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
