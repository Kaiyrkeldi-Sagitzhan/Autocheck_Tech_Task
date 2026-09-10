// Package importer orchestrates the import pipeline: file loading, parsing,
// normalization, deduplication, and upserting into the repository.
package importer

import (
	"context"
	"time"

	"awesomeProject5/internal/domain"
	"awesomeProject5/internal/parser"
	"awesomeProject5/internal/repository"
)

// Importer coordinates a single import run.
type Importer struct {
	repo *repository.Repository
}

// New creates an Importer.
func New(repo *repository.Repository) *Importer {
	return &Importer{repo: repo}
}

// Import processes a 1C export file: parse, validate, upsert, and report.
// It never crashes the whole import because of one malformed row.
func (i *Importer) Import(ctx context.Context, triggerType, fileName string, data []byte) (*domain.ImportReport, error) {
	// Parse CSV.
	records, parseErrors, err := parser.Parse(data)
	if err != nil {
		return nil, err
	}

	// Create import run.
	run := &domain.ImportRun{
		TriggerType: triggerType,
		FileName:    fileName,
		Status:      "running",
		RowsTotal:   len(records) + len(parseErrors),
		StartedAt:   time.Now().UTC(),
	}
	if err := i.repo.CreateImportRun(ctx, run); err != nil {
		return nil, err
	}

	report := &domain.ImportReport{
		File:      fileName,
		RowsTotal: run.RowsTotal,
	}

	// Record parse errors.
	for _, pe := range parseErrors {
		if err := i.repo.RecordImportError(ctx, run.ID, pe.Row, pe.Reason, ""); err != nil {
			return nil, err
		}
		report.Errors = append(report.Errors, pe)
	}

	// Upsert valid records.
	for _, rec := range records {
		car := normalizeCarRecord(rec, fileName)
		created, err := i.repo.UpsertCar(ctx, &car)
		if err != nil {
			// Record error and skip.
			report.Errors = append(report.Errors, domain.RowErrorInfo{
				Row:    rec.RowNumber,
				Reason: "database error: " + err.Error(),
			})
			if err := i.repo.RecordImportError(ctx, run.ID, rec.RowNumber, "database error: "+err.Error(), ""); err != nil {
				return nil, err
			}
			report.Skipped++
			continue
		}
		if created {
			report.Created++
		} else {
			report.Updated++
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

	if err := i.repo.FinishImportRun(ctx, run.ID, run.Status, run); err != nil {
		return nil, err
	}

	return report, nil
}

// normalizeCarRecord converts a parser CarRecord into a domain.Car for persistence.
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
		ImportedAt:   time.Now().UTC(),
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
