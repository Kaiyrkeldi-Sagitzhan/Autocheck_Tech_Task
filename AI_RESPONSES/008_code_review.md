# Code Review Report — autocheck.kz Import Service

**Date:** 2026-09-10  
**Reviewer:** Senior Backend Engineer  
**Scope:** Full backend implementation (parser, importer, repository, scheduler, API, database, Docker)

---

## Executive Summary

The codebase is well-structured with clear separation of concerns. A review identified several HIGH and MEDIUM severity issues. **All MVP-critical issues have been fixed.** The remaining items are lower-priority improvements for future iterations.

---

## Issues Fixed (MVP-Critical)

### 1. No Transaction Atomicity During Import — FIXED
**File:** `internal/importer/importer.go`  
**Severity:** HIGH → RESOLVED

**Problem:** The entire import pipeline executed outside of a database transaction. If the import failed halfway through, partial data was committed.

**Fix:** Wrapped the entire import in a transaction using `repo.BeginTx()` and `repo.WithTx()`. All DB operations (create run, record errors, upsert cars, finish run) now participate in the same transaction. On failure, `defer tx.Rollback()` ensures no partial data is committed. On success, `tx.Commit()` finalizes the import.

**Changes:**
- Added `BeginTx()` and `WithTx()` to `internal/repository/repository.go`
- Changed `Repository.db` to use a `dbHandle` interface supporting both `*sql.DB` and `*sql.Tx`
- Wrapped import DB operations in a transaction in `internal/importer/importer.go`

---

### 2. TOCTOU Race Condition in `UpsertCar` — FIXED
**File:** `internal/repository/repository.go`  
**Severity:** HIGH → RESOLVED

**Problem:** The check-then-insert pattern (`SELECT EXISTS` followed by `INSERT`) was not atomic. Concurrent imports could both see `exists = false` and one would fail with a UNIQUE constraint violation.

**Fix:** Replaced the check-then-insert pattern with an atomic `INSERT` followed by constraint violation handling. If the INSERT fails with a UNIQUE constraint error, the code falls back to `UPDATE`. This eliminates the race window.

**Changes:**
- Removed the `SELECT EXISTS` query
- Added `isConstraintError()` helper to detect UNIQUE constraint violations
- On constraint violation, fetch existing car and perform UPDATE if data changed

---

### 3. Duplicate VINs Within CSV Not Detected — FIXED
**File:** `internal/importer/importer.go`  
**Severity:** HIGH → RESOLVED

**Problem:** If the same VIN appeared multiple times in one CSV, subsequent occurrences would silently update the first. Users received no warning.

**Fix:** Added `detectDuplicateVINs()` function that scans parsed records for duplicate VINs. Duplicates are reported as warnings in the import report (not as hard errors), and all records are still processed normally (first occurrence inserts, subsequent occurrences update).

**Changes:**
- Added `detectDuplicateVINs()` helper in `internal/importer/importer.go`
- Duplicate VIN warnings are added to `report.Errors` with clear messaging
- All records are still processed; duplicates are not skipped

---

### 4. `ImportedAt` Timestamp Overwritten on Every Import — FIXED
**File:** `internal/importer/importer.go`  
**Severity:** HIGH → RESOLVED

**Problem:** `normalizeCarRecord` always set `ImportedAt: time.Now().UTC()`. The guard `if car.ImportedAt.IsZero()` in `UpsertCar` never triggered, so the original import timestamp was lost on every update.

**Fix:** Removed `ImportedAt` from `normalizeCarRecord`. The repository now sets `ImportedAt` only when it is zero (i.e., on INSERT), preserving the original import timestamp for existing cars.

**Changes:**
- Removed `ImportedAt: time.Now().UTC()` from `normalizeCarRecord()`
- `UpsertCar` now correctly sets `ImportedAt` only on new inserts

---

### 5. `file.Read(buf)` May Not Read Entire File — FIXED
**File:** `internal/api/router.go`  
**Severity:** MEDIUM → RESOLVED

**Problem:** `io.Reader.Read` is not guaranteed to fill the buffer in one call. Large file uploads could be silently truncated.

**Fix:** Replaced `file.Read(buf)` with `io.ReadFull(file, buf)` which reads exactly `len(buf)` bytes or returns an error.

**Changes:**
- Added `"io"` import to `internal/api/router.go`
- Changed `file.Read(buf)` to `io.ReadFull(file, buf)`

---

### 6. Scheduler Skips DB Record on File Read Failure — FIXED
**File:** `internal/scheduler/scheduler.go`  
**Severity:** MEDIUM → RESOLVED

**Problem:** If `os.ReadFile` failed (e.g., file missing), the scheduler returned without creating an import run record, leaving no audit trail.

**Fix:** Added optional `repo` field to `Scheduler` via `WithRepo()` option. When file read fails and repo is available, a failed import run is created in the database before returning.

**Changes:**
- Added `repo *repository.Repository` field to `Scheduler` struct
- Added `WithRepo()` option function
- On file read failure, create a failed `ImportRun` record if repo is set
- Updated `cmd/server/main.go` to pass `repo` to scheduler via `WithRepo(repo)`

---

### 7. Wrong HTTP Status Code for `ImportInProgressError` — FIXED
**File:** `internal/api/router.go`  
**Severity:** MEDIUM → RESOLVED

**Problem:** `ImportInProgressError` returned `500 Internal Server Error`, which is misleading. An import already in progress is a client conflict.

**Fix:** Added type assertion to detect `*importer.ImportInProgressError` and return `409 Conflict` instead of `500`.

**Changes:**
- Added type check: `if _, ok := err.(*importer.ImportInProgressError); ok`
- Returns `http.StatusConflict` for in-progress imports

---

### 8. No BOM Handling in CSV Parser — FIXED
**File:** `internal/parser/parser.go`  
**Severity:** MEDIUM → RESOLVED

**Problem:** UTF-8 BOM (`EF BB BF`) in CSV exports from Excel caused the first column header to be unmapped, breaking VIN validation for all rows.

**Fix:** Strip the BOM from input bytes before creating the CSV reader.

**Changes:**
- Added BOM detection and stripping at the start of `Parse()`
- Converts bytes to string, checks for BOM prefix, strips it if present

---

## Remaining Issues (Lower Priority for MVP)

These issues were identified but are not critical for MVP deployment:

| # | Issue | Severity | File |
|---|-------|----------|------|
| 9 | CORS wildcard origin (`*`) | LOW | `internal/api/router.go` |
| 10 | No authentication/authorization | LOW | `internal/api/router.go` |
| 11 | Frontend not embedded in Docker image | LOW | `Dockerfile` |
| 12 | Unused `IMPORT_INTERVAL` env var | LOW | `docker-compose.yml` |
| 13 | Cron parser inconsistency | LOW | `internal/config/config.go` |
| 14 | `carsEqual` does not compare `SourceFile` | LOW | `internal/repository/repository.go` |
| 15 | `parseInt` does not handle comma separators | LOW | `internal/parser/parser.go` |
| 16 | `trimSpace` does not handle Unicode whitespace | LOW | `internal/importer/importer.go` |
| 17 | Database path concatenation | LOW | `internal/database/database.go` |
| 18 | No migration versioning | LOW | `internal/database/migrate.go` |
| 19 | `writeJSON` does not handle pre-written headers | LOW | `internal/api/router.go` |
| 20 | `handleImport` returns `400` for file read errors | LOW | `internal/api/router.go` |

---

## Test Results

All tests pass after fixes:

```
ok  	awesomeProject5/internal/api	0.764s
ok  	awesomeProject5/internal/importer	1.271s
ok  	awesomeProject5/internal/parser	(cached)
ok  	awesomeProject5/internal/repository	(cached)
ok  	awesomeProject5/internal/scheduler	16.574s
```

---

## Files Modified

1. `internal/repository/repository.go` — Added `dbHandle` interface, `BeginTx()`, `WithTx()`, fixed `UpsertCar` TOCTOU
2. `internal/importer/importer.go` — Added transaction wrapping, duplicate VIN detection, fixed `ImportedAt`
3. `internal/parser/parser.go` — Added BOM stripping
4. `internal/api/router.go` — Fixed `io.ReadFull`, HTTP 409 for import-in-progress
5. `internal/scheduler/scheduler.go` — Added `WithRepo()` option, failed run recording
6. `cmd/server/main.go` — Passed `repo` to scheduler
