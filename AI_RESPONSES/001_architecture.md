# Autocheck.kz — Car Inventory Import Microservice: MVP Architecture

## 1. Overview

A single Go service that imports car inventory "1C-style" export files (CSV/XML), normalizes and upserts them into SQLite by VIN, exposes a REST API, and is served alongside a small React frontend. Everything runs via Docker Compose. Designed to be built in a few hours.

> **Conscious MVP tradeoff:** this is architecturally a **modular monolith wearing a microservice's clothes** — one deployable unit containing import, storage, API, and frontend. That is the right call for an hours-scale MVP, but it is a deliberate choice, not an accident. The package boundaries (`parser`, `importer`, `repository`, `api`) are drawn so that any of them can be extracted into a true separate service later without rewriting business logic.

## 2. Architectural Decisions (and why)

| Decision | Rationale |
|---|---|
| **Single Go binary (modular monolith)** | MVP scope; one service handles import, storage, API. No message queues, no multiple services. Explicit tradeoff — see note in §1. |
| **SQLite** | Zero-ops, file-based, perfect for MVP volumes (thousands–tens of thousands of cars). WAL mode for concurrent reads during writes. Trivial to swap for Postgres later via a thin repository layer. |
| **Standard library `net/http` (Go 1.22+ routing)** | No framework needed; fewer dependencies, faster to build, easy to review. |
| **CSV as the primary 1C export format** | 1C exports are commonly CSV/TXT with delimiter + encoding quirks (windows-1251). Parser is isolated in one package so XML/JSON can be added later. |
| **Scheduled job inside the process** | A simple `time.Ticker` goroutine (or cron-style interval from env) instead of external cron/queue. One less moving part. |
| **Import run state recorded in DB (`import_runs`)** | The DB — not an in-process mutex — is the source of truth for "is an import running" and run history. Benefits: (a) `GET /import/status` reads it with no extra plumbing, (b) it survives multiple replicas, (c) audit history for free. A mutex is still used, but only to prevent goroutine races within a process. |
| **Failed rows persisted (`import_errors`) + raw row kept (`defects_raw`)** | 1C data is messy (encoding, garbage numerics, unknown columns). The HTTP import report vanishes once the response is gone; persisted errors and the original raw row let you debug exactly what the export contained. Cost: one `CREATE TABLE` and one insert per failed row. |
| **Manual trigger via HTTP endpoint** | Same import function invoked by scheduler or webhook — one code path, idempotent. |
| **Upsert by VIN** | `INSERT ... ON CONFLICT(vin) DO UPDATE` in SQLite. Guarantees idempotency for repeated imports of the same file. |
| **React served as static files by the same Go binary** | Build the frontend, embed with `go:embed`, serve at `/`. No nginx needed in MVP. |
| **Docker Compose: 2 services** | `app` (Go, multi-stage build) + optional `frontend-dev` (Vite dev server) — or just the single `app` for production-style run. |

## 3. Components

```
autocheck/
├── cmd/server/main.go          # entrypoint: config, router, scheduler start
├── internal/
│   ├── api/                    # HTTP handlers (cars, import trigger, health)
│   ├── importer/               # file loading, parsing, normalization, upsert orchestration
│   ├── parser/                 # 1C CSV parsing (encoding, delimiters, column mapping)
│   ├── repository/             # SQLite access (cars table, upsert, queries)
│   ├── scheduler/              # ticker-based scheduled import
│   └── config/                 # env config
├── web/                        # React app (Vite), built output embedded
├── data/imports/               # watched directory for 1C export files
├── migrations/                 # SQL schema
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

Component responsibilities:

- **parser** — pure functions: bytes → `[]CarRecord` + parse errors. Handles encoding (windows-1251/UTF-8), delimiter sniffing, header mapping, field validation (VIN regex/length 17).
- **importer** — orchestrates: read file(s) from `data/imports/` (or uploaded bytes), call parser, dedupe within batch, upsert into repository, return import report (imported / updated / skipped / failed).
- **repository** — all SQL. Upsert by VIN, list with filters/pagination, get by VIN.
- **api** — REST handlers, JSON responses, basic validation.
- **scheduler** — runs import every `IMPORT_INTERVAL` (e.g. 5m); uses an in-process mutex only to prevent goroutine races; the authoritative "running" state lives in the `import_runs` table.
- **web** — React table of cars with search/filter; fetches `/api/cars`.

## 4. Data Flow

```
1C export file (CSV)
        │
        ├── (a) dropped into data/imports/  ──► scheduler (ticker) picks up
        ├── (b) POST /api/v1/import  (multipart upload or path trigger)
        ▼
   importer.Import()
        │  1. read file bytes
        │  2. parser.Parse() → []CarRecord (+ row errors)
        │  3. normalize: trim, uppercase VIN, parse ints, map defect codes
        │  4. dedupe by VIN within batch (last wins)
        ▼
   repository.BeginImportRun()   (import_runs: status='running')
         ▼
   repository.UpsertCars()   (batched transactions, ON CONFLICT(vin))
         ▼
   SQLite (cars table)  +  import_errors (failed rows, raw payload)
         ▼
   repository.FinishImportRun()  (status='success'/'failed', counts)
        ▼
   GET /api/v1/cars  ◄──── React frontend (table, search, pagination)
```

Import is idempotent: re-importing the same file updates existing rows by VIN and bumps `updated_at`.

## 5. Database Schema

```sql
CREATE TABLE IF NOT EXISTS cars (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    vin           TEXT NOT NULL UNIQUE,          -- normalized: uppercase, trimmed
    brand         TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    year          INTEGER,                       -- nullable: may be missing in export
    mileage_km    INTEGER,
    price         INTEGER,                       -- whole-unit currency, see note below
    currency      TEXT NOT NULL DEFAULT 'KZT',   -- companion to price; MVP assumes KZT-only
    color         TEXT,
    engine        TEXT,                          -- e.g. "2.0 petrol 150hp"
    transmission  TEXT,                          -- AT / MT / CVT
    body_type     TEXT,
    defects       TEXT NOT NULL DEFAULT '[]',    -- JSON array of normalized defect strings/codes
    defects_raw   TEXT NOT NULL DEFAULT '',      -- original defect field as it appeared in the 1C export
    status        TEXT NOT NULL DEFAULT 'active',-- active / sold / reserved
    source_file   TEXT NOT NULL DEFAULT '',      -- provenance
    imported_at   DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cars_brand_model ON cars(brand, model);
CREATE INDEX IF NOT EXISTS idx_cars_status      ON cars(status);

CREATE TABLE IF NOT EXISTS import_runs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger_type  TEXT NOT NULL,                 -- 'scheduled' | 'manual'
    file_name     TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL,                 -- 'running' | 'success' | 'failed'
    rows_total    INTEGER NOT NULL DEFAULT 0,
    created       INTEGER NOT NULL DEFAULT 0,
    updated       INTEGER NOT NULL DEFAULT 0,
    skipped       INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at    DATETIME NOT NULL,
    finished_at   DATETIME
);

CREATE TABLE IF NOT EXISTS import_errors (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id        INTEGER NOT NULL REFERENCES import_runs(id),
    row_number    INTEGER NOT NULL,
    reason        TEXT NOT NULL,
    raw_row       TEXT NOT NULL DEFAULT '',      -- original line/row content for debugging
    created_at    DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_import_errors_run ON import_errors(run_id);
```

Notes:
- `defects` stored as a JSON string — MVP-appropriate; a separate `car_defects` table is the normalized alternative if filtering by defect becomes a requirement. `defects_raw` preserves the untouched 1C field so normalization bugs can be diagnosed without re-obtaining the export.
- `price` is a whole-unit integer with a `currency` companion column defaulting to `'KZT'`. MVP assumes KZT-only; the column exists so multi-currency listings in real 1C exports don't require a migration later.
- `year`, `mileage_km`, `price` nullable/zero-able because 1C exports often have blanks.
- `import_runs` is the authoritative record of import state and history; `import_errors` persists per-row failures (with the raw row) that would otherwise be lost when the HTTP import report response is gone.
- Upsert: `INSERT ... ON CONFLICT(vin) DO UPDATE SET ... , updated_at = CURRENT_TIMESTAMP`.

## 6. API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/cars` | List cars. Query params: `page`, `page_size`, `brand`, `model`, `vin` (partial), `status`, `min_year`, `max_mileage`. Returns `{items, total, page, page_size}`. |
| `GET` | `/api/v1/cars/{vin}` | Single car by VIN. 404 if absent. |
| `POST` | `/api/v1/import` | Manual import trigger. Accepts multipart file upload **or** `{"file": "name.csv"}` to import a file already in `data/imports/`. Returns import report. |
| `GET` | `/api/v1/import/status` | Reads `import_runs`: current/last run status, counts, timing. |
| `GET` | `/api/v1/import/runs/{id}/errors` | Paginated `import_errors` for a run (row number, reason, raw row) for debugging messy exports. |
| `GET` | `/healthz` | Liveness probe. |
| `GET` | `/*` | Serves embedded React frontend. |

Import report example:
```json
{
  "file": "inventory_2026-09-10.csv",
  "rows_total": 512,
  "created": 300,
  "updated": 200,
  "skipped": 10,
  "errors": [{"row": 42, "reason": "invalid VIN length"}]
}
```

## 7. Edge Cases

1. **Encoding** — 1C often exports windows-1251. Detect BOM; fall back to `golang.org/x/text/encoding/charmap` decode.
2. **Delimiter variance** — `;` vs `,` vs tab. Sniff from header line or make it configurable.
3. **Invalid/missing VIN** — skip row, record error, don't fail the whole import. VIN normalized (uppercase, strip spaces/dashes); validate 17 chars, no I/O/Q.
4. **Duplicate VINs within one file** — last occurrence wins; dedupe before upsert.
5. **Empty numeric fields / garbage ("120 000 км")** — tolerant parsing; store NULL on failure.
6. **Cars removed from inventory** — 1C exports are snapshots. MVP: mark rows not present in the latest full import as `status='sold'` (only for full-file imports, not partial uploads). Configurable flag.
7. **Concurrent imports** — scheduler + manual trigger overlapping. In-process `sync.Mutex`/`TryLock` prevents goroutine races; the authoritative check is a `status='running'` row in `import_runs` (inserted transactionally), so a manual trigger returns 409 if an import is in progress — including one started by another replica.
8. **Large files** — stream row-by-row; batch upserts (e.g. 500 rows per transaction) to keep memory and lock time low.
9. **Malformed rows / extra columns** — map by header names, ignore unknown columns, log them.
10. **SQLite write contention with API reads** — WAL mode; reads don't block.
11. **File partially written while scheduler reads it** — only pick up files stable for N seconds, or process `.done`/rename pattern (`.csv` → process → move to `processed/`).
12. **Timezones** — store UTC; 1C dates may be `DD.MM.YYYY`; parse explicitly.

## 8. Docker Compose

```yaml
services:
  app:
    build: .
    ports: ["8080:8080"]
    environment:
      - DB_PATH=/data/autocheck.db
      - IMPORT_DIR=/data/imports
      - IMPORT_INTERVAL=300s
    volumes:
      - app-data:/data
volumes:
  app-data:
```

Single service; SQLite file and import directory on a named volume. Frontend is built in the multi-stage Dockerfile and embedded into the Go binary.

## 9. Build Order (suggested, ~hours)

1. Schema + repository + upsert (with a tiny seed test).
2. Parser for sample 1C CSV.
3. Importer + manual `POST /api/v1/import`.
4. Scheduler.
5. `GET /api/v1/cars` + filters.
6. React table page.
7. Dockerfile + compose; end-to-end test with a sample file.
