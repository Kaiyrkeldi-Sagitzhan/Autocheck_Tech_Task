// Package importer orchestrates the import pipeline: file loading, parsing,
// normalization, deduplication, and upserting into the repository.
package importer

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"awesomeProject5/internal/domain"
	"awesomeProject5/internal/parser"
	"awesomeProject5/internal/repository"
)

// ImportRequest carries all parameters for a single import run.
type ImportRequest struct {
	TriggerType string // "manual" | "scheduled"
	TriggeredBy string // e.g. "api", "cron"
	FileName    string
	Data        []byte
}

// Importer coordinates a single import run.
// It is safe for concurrent use: only one Import call may be active at a time.
type Importer struct {
	repo    *repository.Repository
	running atomic.Bool
	runInfo *importRunInfo
	runMu   sync.RWMutex // protects runInfo reads/writes
}

type importRunInfo struct {
	startedAt   time.Time
	triggeredBy string
	triggerType string
}

// New creates an Importer.
func New(repo *repository.Repository) *Importer {
	return &Importer{repo: repo}
}

// Import processes a 1C export file: parse, validate, upsert, and report.
// It never crashes the whole import because of one malformed row.
// Only one Import call may be active at a time across all callers.
func (i *Importer) Import(ctx context.Context, req ImportRequest) (*domain.ImportReport, error) {
	if !i.running.CompareAndSwap(false, true) {
		i.runMu.RLock()
		info := i.runInfo
		i.runMu.RUnlock()
		return nil, &ImportInProgressError{
			StartedAt:   info.startedAt,
			TriggeredBy: info.triggeredBy,
			TriggerType: info.triggerType,
		}
	}
	defer i.running.Store(false)

	i.runMu.Lock()
	i.runInfo = &importRunInfo{
		startedAt:   time.Now().UTC(),
		triggeredBy: req.TriggeredBy,
		triggerType: req.TriggerType,
	}
	i.runMu.Unlock()
	defer func() {
		i.runMu.Lock()
		i.runInfo = nil
		i.runMu.Unlock()
	}()

	// Parse CSV.
	records, parseErrors, err := parser.Parse(req.Data)
	if err != nil {
		return nil, err
	}

	// Detect duplicate VINs within the same file (for warnings).
	duplicateVINs := detectDuplicateVINs(records)

	// Start a transaction to ensure atomicity.
	tx, err := i.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	txRepo := i.repo.WithTx(tx)

	// Create import run.
	run := &domain.ImportRun{
		TriggerType: req.TriggerType,
		TriggeredBy: req.TriggeredBy,
		FileName:    req.FileName,
		Status:      "running",
		RowsTotal:   len(records) + len(parseErrors),
		StartedAt:   time.Now().UTC(),
	}
	if err := txRepo.CreateImportRun(ctx, run); err != nil {
		return nil, err
	}

	report := &domain.ImportReport{
		File:      req.FileName,
		RowsTotal: run.RowsTotal,
	}

	// Record parse errors.
	for _, pe := range parseErrors {
		if err := txRepo.RecordImportError(ctx, run.ID, pe.Row, pe.Reason, ""); err != nil {
			return nil, err
		}
		report.Errors = append(report.Errors, pe)
	}

	// Add duplicate VIN warnings to the report.
	for _, rows := range duplicateVINs {
		rowStrs := make([]string, len(rows))
		for i, r := range rows {
			rowStrs[i] = strconv.Itoa(r)
		}
		reason := "warning: duplicate VIN in file (rows: " + strings.Join(rowStrs, ", ") + ")"
		for _, row := range rows {
			report.Errors = append(report.Errors, domain.RowErrorInfo{
				Row:    row,
				Reason: reason,
			})
		}
	}

	// Upsert valid records.
	for _, rec := range records {

		car := normalizeCarRecord(rec, req.FileName)
		created, changed, err := txRepo.UpsertCar(ctx, &car)
		if err != nil {
			// Record error and skip.
			report.Errors = append(report.Errors, domain.RowErrorInfo{
				Row:    rec.RowNumber,
				Reason: "database error: " + err.Error(),
			})
			if err := txRepo.RecordImportError(ctx, run.ID, rec.RowNumber, "database error: "+err.Error(), ""); err != nil {
				return nil, err
			}
			report.Skipped++
			continue
		}
		if created {
			report.Created++
		} else if changed {
			report.Updated++
		} else {
			// Existing VIN with identical data — no DB write needed.
			report.Skipped++
		}
	}

	// Finish import run.
	run.Created = report.Created
	run.Updated = report.Updated
	run.Skipped = report.Skipped
	run.ErrorMessage = ""
	if len(report.Errors) > 0 {
		run.Status = "failed"
		run.ErrorMessage = "import completed with errors"
	} else {
		run.Status = "success"
	}
	now := time.Now().UTC()
	run.FinishedAt = &now

	if err := txRepo.FinishImportRun(ctx, run.ID, run.Status, run); err != nil {
		return nil, err
	}

	// Commit the transaction.
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return report, nil
}

// detectDuplicateVINs returns a map of VIN -> row numbers for VINs that appear
// more than once in the records slice.
func detectDuplicateVINs(records []domain.CarRecord) map[string][]int {
	vinCounts := make(map[string][]int)
	for _, rec := range records {
		vinCounts[rec.VIN] = append(vinCounts[rec.VIN], rec.RowNumber)
	}

	duplicates := make(map[string][]int)
	for vin, rows := range vinCounts {
		if len(rows) > 1 {
			duplicates[vin] = rows
		}
	}
	return duplicates
}

// normalizeCarRecord converts a parser CarRecord into a domain.Car for persistence.
// ImportedAt is left as zero value; the repository sets it on INSERT.
func normalizeCarRecord(rec domain.CarRecord, sourceFile string) domain.Car {
	car := domain.Car{
		VIN:          rec.VIN,
		Brand:        rec.Brand,
		Model:        rec.Model,
		Year:         rec.Year,
		MileageKm:    rec.MileageKm,
		Price:        rec.Price,
		Currency:     rec.Currency,
		Color:        rec.Color,
		Engine:       rec.Engine,
		Transmission: rec.Transmission,
		BodyType:     rec.BodyType,
		DefectsRaw:   rec.DefectsRaw,
		Status:       rec.Status,
		SourceFile:   sourceFile,
		UpdatedAt:    time.Now().UTC(),
	}

	// Normalize defects: split by comma if present, otherwise use raw.
	if rec.DefectsRaw != "" {
		car.Defects = splitDefects(rec.DefectsRaw)
	} else {
		car.Defects = []string{}
	}

	return car
}

// splitDefects splits a raw defects string by comma and trims each part.
func splitDefects(raw string) []string {
	parts := []string{}
	current := ""
	for _, c := range raw {
		if c == ',' {
			trimmed := trimSpace(current)
			if trimmed != "" {
				parts = append(parts, trimmed)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	trimmed := trimSpace(current)
	if trimmed != "" {
		parts = append(parts, trimmed)
	}
	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
