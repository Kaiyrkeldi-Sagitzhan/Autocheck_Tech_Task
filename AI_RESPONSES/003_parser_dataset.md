# 1C Automobile Export Dataset and Parser Implementation

**Date:** 2026-09-10 (Asia/Almaty)
**Mode:** 💻 Code
**Basis:** [AI_RESPONSES/001_architecture.md](../001_architecture.md), [AI_RESPONSES/002_project_structure.md](../002_project_structure.md)

## Overview

This document describes the implementation of the 1C-like automobile export dataset and parser. The work includes a realistic CSV dataset with 1000+ records, a deterministic generator, a production-grade CSV parser, comprehensive unit tests, and a Docker runtime fix.

## Files Created

| File | Purpose |
|------|---------|
| `internal/parser/parser.go` | CSV parser implementation |
| `internal/parser/parser_test.go` | Unit tests for the parser |
| `cmd/gen-dataset/main.go` | Deterministic Go dataset generator |
| `scripts/generate_dataset.py` | Python fallback generator (used for initial dataset) |
| `data/imports/partner_1c_export.csv` | Generated 1C export dataset (1000 rows) |

## Files Modified

| File | Changes |
|------|---------|
| `internal/domain/car.go` | Added `Status` and `UpdatedAt` fields to `CarRecord` |
| `Dockerfile` | Fixed SQLite path to `/app/data`, created directories with proper ownership |
| `docker-compose.yml` | Updated volume mount and environment variables to match new path |

## Dataset Details

- **File:** `data/imports/partner_1c_export.csv`
- **Records:** 1000 data rows + 1 header
- **Delimiter:** Semicolon (`;`)
- **Encoding:** UTF-8
- **Line endings:** CRLF (`\r\n`)
- **Deterministic:** Yes (seed=42)

### Columns

| Column | Description |
|--------|-------------|
| VIN | 17-character Vehicle Identification Number |
| Brand | Manufacturer (e.g., Toyota, BMW, Hyundai) |
| Model | Model name (e.g., Camry, X5, Tucson) |
| Year | Production year (2000–2025) |
| MileageKm | Odometer reading in kilometers |
| Price | Price in whole currency units |
| Currency | Currency code (default: KZT) |
| Color | Exterior color (Russian names) |
| Engine | Engine description (e.g., "2.0 бензин 150 л.с.") |
| Transmission | Transmission type (AT, MT, CVT, DCT, AMT) |
| BodyType | Body style (Седан, Хэтчбек, SUV, etc.) |
| DefectsRaw | Original defect field from 1C export |
| Status | active / sold / reserved |
| UpdatedAt | Date string (YYYY-MM-DD) |

### Problematic Records (Intentionally Included)

| Row | Issue | Description |
|-----|-------|-------------|
| 5 | Missing VIN | Empty VIN field |
| 50 | Invalid mileage | Non-numeric value ("abc") |
| 100 | Invalid year | Out-of-range year (1800) |
| 200 | Missing brand | Empty brand field |
| 500 | Malformed columns | Too few columns in row |
| 750 | Missing model | Empty model field |

## Parser Implementation

### Architecture

The parser is a pure function with no I/O or database dependencies:

```go
func Parse(data []byte) ([]domain.CarRecord, []domain.RowErrorInfo, error)
```

### Features

- **Semicolon delimiter** — configured via `csv.Reader.Comma = ';'`
- **Header mapping** — columns mapped by normalized (lowercased, trimmed) header names
- **Whitespace trimming** — all fields trimmed automatically
- **VIN validation** — 17 characters, no I/O/Q
- **Safe numeric parsing** — `strconv.Atoi` with space/non-breaking-space stripping; returns `nil` on failure
- **Year validation** — fatal error if out of 1900–2100 range
- **Mileage validation** — fatal error if negative
- **Row-level errors** — collected in `[]RowErrorInfo` with row number and reason
- **Resilient** — continues processing valid rows when another row is invalid
- **Default currency** — defaults to "KZT" when missing
- **Status mapping** — case-insensitive mapping to `CarStatus` constants

### Validation Rules

| Field | Rule | Severity |
|-------|------|----------|
| VIN | Required, 17 chars, no I/O/Q | Fatal |
| Brand | Required, non-empty | Fatal |
| Model | Required, non-empty | Fatal |
| Year | Optional, 1900–2100 if present | Fatal if invalid |
| MileageKm | Optional, >= 0 if present | Fatal if invalid |
| Price | Optional, any integer | Non-fatal (nil on parse failure) |
| Currency | Optional, defaults to KZT | Non-fatal |
| Status | Optional, defaults to active | Non-fatal |

## Unit Tests

13 test functions covering:

- Valid CSV parsing with all fields
- Missing VIN
- Invalid VIN (too short, too long, invalid characters)
- Invalid mileage (non-numeric → nil, no error)
- Invalid year (out of range → fatal; non-numeric → nil)
- Missing required fields (brand, model)
- Malformed columns (too few fields)
- Mixed valid and invalid rows
- Empty file
- Header-only file
- Whitespace trimming
- Default currency fallback
- Status mapping (case-insensitive)
- Negative mileage validation
- Row number tracking
- Large file (1000 rows)

## Docker Runtime Fix

### Problem

SQLite error 14 (`SQLITE_CANTOPEN`) occurred during container startup:

```
migrations: apply schema: unable to open database file: out of memory (14)
```

### Root Cause

The container runs as non-root user `app`, but the Docker volume was mounted at `/data` which is owned by `root`. The application's `database.Open()` function calls `os.MkdirAll("/data", 0o755)` to ensure the directory exists, but this fails because user `app` lacks write permission to `/data`.

### Fix

1. Changed database path from `/data/autocheck.db` to `/app/data/autocheck.db`
2. Created `/app/data/imports` directories in the Dockerfile with `app` user ownership before switching to the `app` user
3. Updated volume mount to `/app/data`
4. Updated `docker-compose.yml` environment variables to match

### Verification

```bash
docker compose down -v
docker compose build --no-cache
docker compose up -d
docker logs awesomeproject5-app-1
# Output: 2026/09/10 12:56:27 listening on :8080
```

## Commands

### Generate Dataset

```bash
# Using Python (already done)
python3 scripts/generate_dataset.py

# Using Go generator
go run ./cmd/gen-dataset
```

### Run Tests

```bash
go test ./...
```

### Format Code

```bash
gofmt -w internal/ cmd/ scripts/
```

### Docker

```bash
docker compose down -v
docker compose build --no-cache
docker compose up -d
```

## Status

- Parser implementation complete and tested.
- Dataset generated with 1000 records + 6 problematic rows.
- Docker runtime issue resolved.
- All tests pass.
- Code formatted with gofmt.
