# Autocheck.kz — Initial Project Structure

**Date:** 2026-09-10 (Asia/Almaty)
**Mode:** 💻 Code
**Basis:** [AI_RESPONSES/001_architecture.md](001_architecture.md)

## Overview

This document describes the initial project structure created from the approved
architecture. The scope is deliberately minimal: package layout, scaffolding,
configuration, database schema, Docker setup, and a React frontend shell. No
business logic is implemented — every runtime package is a stub with TODOs for
later phases.

## Component Map

```
awesomeProject5/
├── .dockerignore              # Docker build context exclusions
├── .env.example               # Documented runtime environment variables
├── .gitignore                 # Local env, build output, SQLite files, frontend artifacts
├── Dockerfile                 # Multi-stage frontend/backend/runtime build
├── docker-compose.yml         # App service + named volume
├── go.mod / go.sum            # Go module: awesomeProject5, modernc.org/sqlite
├── README.md                  # Skeleton: structure, config, run commands, planned API
├── AI_LOGS.md                 # Chronological task log
├── AI_RESPONSES/
│   ├── 001_architecture.md    # Approved architecture proposal
│   └── 002_project_structure.md  # This file
├── cmd/
│   └── server/
│       └── main.go            # Entrypoint: config → db → migrate → router → shutdown
├── data/
│   └── imports/
│       └── .gitkeep           # Preserves the watched import directory
├── internal/
│   ├── api/
│   │   └── router.go          # HTTP mux + handlers (health, not-implemented stubs)
│   ├── config/
│   │   └── config.go          # Environment-based Config loader
│   ├── database/
│   │   ├── database.go        # SQLite Open() with WAL, busy_timeout, foreign_keys
│   │   ├── migrate.go         # Migrate() entrypoint
│   │   └── schema.go          # Embedded schemaSQL constant
│   ├── domain/
│   │   └── car.go             # Car, CarRecord, ImportRun, ImportError, ImportReport
│   ├── importer/
│   │   └── importer.go        # Importer scaffold (no logic yet)
│   ├── parser/
│   │   └── parser.go          # Parse() stub returning nil
│   ├── repository/
│   │   └── repository.go      # Repository scaffold (db *sql.DB)
│   └── scheduler/
│       └── scheduler.go       # Ticker-based Scheduler with Start/Stop
├── migrations/
│   └── 001_schema.sql         # Human-readable SQLite schema (cars, import_runs, import_errors)
└── web/
    ├── index.html             # React shell HTML
    ├── package.json           # Vite + React manifest
    ├── vite.config.js         # Dev proxy to :8080
    ├── package-lock.json
    ├── src/
    │   ├── main.jsx           # React entrypoint
    │   ├── App.jsx            # App shell
    │   └── index.css          # Minimal global styles
    └── dist/                  # Built output (gitignored)
```

## Package Responsibilities

| Package | File | Status |
|---|---|---|
| `cmd/server` | `main.go` | Wires config, database, migrations, router, graceful shutdown |
| `internal/api` | `router.go` | Registers all routes; `/healthz` returns ok, others return 501 |
| `internal/config` | `config.go` | Loads `HTTP_PORT`, `DB_PATH`, `IMPORT_DIR`, `IMPORT_INTERVAL` with defaults |
| `internal/database` | `database.go`, `migrate.go`, `schema.go` | SQLite connection + embedded schema migration |
| `internal/domain` | `car.go` | Shared domain models across layers |
| `internal/importer` | `importer.go` | Empty `Importer` struct scaffold |
| `internal/parser` | `parser.go` | `Parse()` stub returning `nil, nil, nil` |
| `internal/repository` | `repository.go` | `Repository` struct with `db *sql.DB` |
| `internal/scheduler` | `scheduler.go` | `Scheduler` with `time.Ticker` goroutine, `Start`/`Stop` |

## Database Schema

Three tables, matching the approved architecture:

- `cars` — `vin UNIQUE`, upsert by VIN via `ON CONFLICT(vin) DO UPDATE`
- `import_runs` — tracks running/success/failed state per import
- `import_errors` — per-row failures with raw payload for debugging

Indexes on `cars(brand, model)`, `cars(status)`, `import_errors(run_id)`.

## API Endpoints (all stubs)

| Method | Path | Status |
|---|---|---|
| `GET` | `/healthz` | Implemented (returns `{"status":"ok"}`) |
| `GET` | `/api/v1/cars` | 501 stub |
| `GET` | `/api/v1/cars/{vin}` | 501 stub |
| `POST` | `/api/v1/import` | 501 stub |
| `GET` | `/api/v1/import/status` | 501 stub |
| `GET` | `/api/v1/import/runs/{id}/errors` | 501 stub |

## Configuration

All runtime configuration is via environment variables:

| Variable | Default | Description |
|---|---|---|
| `HTTP_PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `data/autocheck.db` | SQLite database file path |
| `IMPORT_DIR` | `data/imports` | Watched directory for 1C export files |
| `IMPORT_INTERVAL` | `5m` | Scheduled import interval |

## Validation Performed

- `go build ./...` — all packages compile cleanly
- `gofmt -l internal/ cmd/` — no formatting issues
- `go vet ./...` — no vet warnings
- `docker compose build app` — Docker image builds successfully
- `npm run build` — React/Vite shell builds successfully
- `docker compose config --quiet` — compose configuration is valid
- `go mod tidy` — go.sum updated with all transitive dependencies

## Status

- Minimal project structure created.
- Business logic intentionally left as stubs/TODOs for later implementation phases.