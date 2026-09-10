# Import Service and VIN-Based Upsert Implementation

**Date:** 2026-09-10 (Asia/Almaty)
**Mode:** 💻 Code
**Basis:** [AI_RESPONSES/001_architecture.md](../001_architecture.md), [AI_RESPONSES/002_project_structure.md](../002_project_structure.md), [AI_RESPONSES/003_parser_dataset.md](../003_parser_dataset.md)

## Overview

This document describes the implementation of the import service and VIN-based upsert logic. The import service connects the CSV parser to the SQLite repository, orchestrating validation, normalization, and persistence.

## Architecture

```
CSV parser → validation → importer → repository → SQLite
```

- **Parser** — pure function: bytes → `[]CarRecord` + parse errors
- **Importer** — orchestrates: parse → normalize → upsert → track errors → return report
- **Repository** — data access: upsert by VIN, import run bookkeeping
- **Domain** — shared models: `Car`, `CarRecord`, `ImportRun`, `ImportError`, `ImportReport`

## Files Modified

| File | Changes |
|------|---------|
| `internal/repository/repository.go` | Added `UpsertCar`, `CreateImportRun`, `RecordImportError`, `FinishImportRun`, `CarByVIN` |
| `internal/importer/importer.go` | Implemented `Import` method and `normalizeCarRecord` |

## Files Created

| File | Purpose |
|------|---------|
| `internal/repository/repository_test.go` | Repository unit tests |
| `internal/importer/importer_test.go` | Importer integration tests |

## Repository Methods

### UpsertCar

```go
func (r *Repository) UpsertCar(ctx context.Context, car *domain.Car) (bool, error)
```

- Checks if VIN exists via `SELECT EXISTS(SELECT 1 FROM cars WHERE vin = ?)`
- If new: `INSERT INTO cars ...` → returns `(true, nil)`
- If existing: `UPDATE cars SET ... WHERE vin = ?` → returns `(false, nil)`
- Never creates duplicates because VIN is UNIQUE
- Serializes `Defects` to JSON for storage
- Sets `ImportedAt` and `UpdatedAt` timestamps

### CreateImportRun

```go
func (r *Repository) CreateImportRun(ctx context.Context, run *domain.ImportRun) error
```

- Inserts a new `import_runs` record
- Sets `run.ID` from `LastInsertId()`

### RecordImportError

```go
func (r *Repository) RecordImportError(ctx context.Context, runID int64, rowNumber int, reason, rawRow string) error
```

- Inserts a row-level failure into `import_errors`

### FinishImportRun

```go
func (r *Repository) FinishImportRun(ctx context.Context, runID int64, status string, run *domain.ImportRun) error
```

- Updates the import run with final status, counts, and `finished_at`

### CarByVIN

```go
func (r *Repository) CarByVIN(ctx context.Context, vin string) (*domain.Car, error)
```

- Retrieves a car by VIN for test verification
- Deserializes `defects` JSON

## Importer Methods

### Import

```go
func (i *Importer) Import(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error)
```

Orchestrates the full import pipeline:

1. **Parse** — calls `parser.Parse(data)` to get records and parse errors
2. **Create import run** — inserts `import_runs` record with status "running"
3. **Record parse errors** — logs each parse error in `import_errors`
4. **Upsert valid records** — for each `CarRecord`:
   - Normalizes to `domain.Car` via `normalizeCarRecord()`
   - Calls `repo.UpsertCar()` → tracks created/updated counts
   - On database error: records error, increments skipped
5. **Finish import run** — updates `import_runs` with final status and counts
6. **Return report** — `ImportReport` with totals and errors

### normalizeCarRecord

Converts a parser `CarRecord` into a persistence-ready `domain.Car`:
- Copies all fields
- Normalizes defects: splits `DefectsRaw` by comma, trims each part
- Sets `ImportedAt` and `UpdatedAt` to current UTC time
- Sets `SourceFile` to the import file name

## Upsert Strategy

VIN is the unique business identifier. The `cars` table schema enforces this:

```sql
CREATE TABLE IF NOT EXISTS cars (
    vin TEXT NOT NULL UNIQUE,
    ...
);
```

The repository uses a pre-check approach:
1. `SELECT EXISTS(SELECT 1 FROM cars WHERE vin = ?)` to detect existing VINs
2. If new → `INSERT`
3. If existing → `UPDATE`

This ensures:
- New VIN creates a new car
- Existing VIN updates the existing car
- Importing the same VIN twice never creates a duplicate

## Import Statistics

The `ImportReport` returns:

| Field | Description |
|-------|-------------|
| `RowsTotal` | Total rows in the file (valid + invalid) |
| `Created` | Number of new cars inserted |
| `Updated` | Number of existing cars updated |
| `Skipped` | Number of rows skipped due to database errors |
| `Errors` | List of `RowErrorInfo` with row number and reason |

## Tests

### Repository Tests

| Test | Description |
|------|-------------|
| `TestUpsertCar_InsertNew` | Inserts a new car, verifies all fields |
| `TestUpsertCar_UpdateExisting` | Updates mileage, price, defects on existing VIN |
| `TestUpsertCar_SameVINTwice_NoDuplicate` | Imports same VIN twice, verifies no duplicate |
| `TestUpsertCar_UpdateMileagePriceDefects` | Specifically tests updating those fields |
| `TestCreateAndFinishImportRun` | Tests import run lifecycle |

### Importer Tests

| Test | Description |
|------|-------------|
| `TestImport_NewCars` | Imports 2 new cars, verifies Created=2 |
| `TestImport_UpdateExistingVIN` | Imports same VIN twice, verifies Updated=1 |
| `TestImport_SameVINTwice_NoDuplicate` | Same VIN in same import, verifies Created=1, Updated=1 |
| `TestImport_WithInvalidRows` | Mixed valid/invalid rows, verifies error reporting |
| `TestImport_EmptyFile` | Empty CSV, verifies graceful handling |
| `TestImport_DefectsNormalization` | Verifies comma-separated defects are split correctly |

## Validation Performed

- `gofmt -l internal/ cmd/ scripts/` — clean
- `go test ./...` — all tests pass:
  - `ok awesomeProject5/internal/importer 0.014s`
  - `ok awesomeProject5/internal/parser 0.009s`
  - `ok awesomeProject5/internal/repository 0.008s`
- `docker compose down -v && docker compose build --no-cache && docker compose up -d` — container starts successfully

## Status

- ✅ Import service implemented and tested
- ✅ VIN-based upsert working correctly
- ✅ Duplicate VIN prevention verified
- ✅ Import statistics (created, updated, skipped, errors) implemented
- ✅ Import service independent from HTTP handlers
- ✅ Database transactions used via `ExecContext`
- ✅ Parser validation errors handled without aborting import
- ✅ All tests pass
