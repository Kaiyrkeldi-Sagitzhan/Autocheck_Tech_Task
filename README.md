# Autocheck.kz — Car Inventory Import Microservice

A single Go service that imports car inventory "1C-style" export files (CSV/XML),
normalizes and upserts them into SQLite by VIN, exposes a REST API, and serves
a small React frontend. Everything runs via Docker Compose.

> Architecture: see [AI_RESPONSES/001_architecture.md](AI_RESPONSES/001_architecture.md).

This repository is an initial scaffold only. Runtime packages are present, but
import, parsing, persistence, and API business logic remain stubbed for later phases.

## Project Structure

```
cmd/server/          # entrypoint: config, database, router
internal/
  api/               # HTTP handlers (cars, import trigger, health)
  importer/          # file loading, parsing, normalization, upsert orchestration
  parser/            # 1C CSV parsing (encoding, delimiters, column mapping)
  repository/        # SQLite access (cars table, upsert, queries)
  scheduler/         # ticker-based scheduled import
  config/            # env config
  domain/            # domain models (Car, ImportRun, ImportError, ...)
  database/          # SQLite connection + migrations
web/                 # React app (Vite), build output intended for embedding
migrations/          # SQL schema
data/imports/        # watched directory for 1C export files
.dockerignore        # Docker build context exclusions
```

## Configuration

All configuration is via environment variables (see [.env.example](.env.example)):

| Variable           | Default              | Description                     |
|--------------------|----------------------|---------------------------------|
| `HTTP_PORT`        | `8080`               | HTTP listen port                |
| `DB_PATH`          | `data/autocheck.db`  | SQLite database file path       |
| `IMPORT_DIR`       | `data/imports`       | Watched directory for exports   |
| `IMPORT_INTERVAL`  | `5m`                 | Scheduled import interval       |

## Running

### Docker Compose

```sh
docker compose up --build
```

### Locally

```sh
go run ./cmd/server
```

## API (planned)

| Method | Path                                  | Description                    |
|--------|---------------------------------------|--------------------------------|
| GET    | `/api/v1/cars`                        | List cars (filters, pagination)|
| GET    | `/api/v1/cars/{vin}`                  | Single car by VIN              |
| POST   | `/api/v1/import`                      | Manual import trigger          |
| GET    | `/api/v1/import/status`               | Current/last import run status |
| GET    | `/api/v1/import/runs/{id}/errors`     | Failed rows for a run          |
| GET    | `/healthz`                            | Liveness probe                 |

## Status

Project skeleton only — business logic is implemented in later phases.
See [AI_LOGS.md](AI_LOGS.md) for progress.
