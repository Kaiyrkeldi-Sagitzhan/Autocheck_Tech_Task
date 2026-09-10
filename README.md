# Autocheck.kz — Car Inventory Import Microservice

A Go microservice that imports car inventory "1C-style" export files (CSV), normalizes and upserts them into SQLite by VIN, exposes a REST API, and serves a React frontend.

> Architecture details: see [AI_RESPONSES/001_architecture.md](AI_RESPONSES/001_architecture.md).

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Clone the Repository](#clone-the-repository)
3. [Quick Start (Docker Compose)](#quick-start-docker-compose)
4. [Local Development](#local-development)
5. [Configuration](#configuration)
6. [Running Tests](#running-tests)
7. [API Reference](#api-reference)
8. [Frontend Development](#frontend-development)
9. [Project Structure](#project-structure)
10. [Troubleshooting](#troubleshooting)

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| **Git** | any | clone repository |
| **Docker** | 20.10+ | containerized run (recommended) |
| **Docker Compose** | v2+ | multi-service orchestration |
| **Go** | 1.22+ | local backend development |
| **Node.js** | 18+ | local frontend development |
| **npm** | 9+ | frontend package manager |

> **Tip:** If you only want to run the app, Docker Compose is enough — no Go or Node.js needed.

---

## Clone the Repository

```sh
git clone https://github.com/<your-org>/autocheck.kz.git
cd autocheck.kz
```

---

## Quick Start (Docker Compose)

This is the fastest way to get the entire stack running.

### 1. Copy environment file

```sh
cp .env.example .env
```

Edit `.env` if you need to change ports or paths (defaults work for most cases).

### 2. Start the stack

```sh
docker compose up --build
```

The backend will be available at `http://localhost:8080`.

### 3. Verify it works

```sh
curl http://localhost:8080/api/health
# Expected: {"status":"ok"}
```

Open the frontend by navigating to `http://localhost:8080` in your browser.

> **Note:** The Docker image builds the React frontend and embeds it into the Go binary. There is no separate frontend container in production mode.

### 4. Stop the stack

```sh
docker compose down
```

Data persists in the Docker volume `app-data`. To wipe data:

```sh
docker compose down -v
```

---

## Local Development

### Backend (Go)

#### 1. Install dependencies

```sh
go mod download
```

#### 2. Configure environment

```sh
cp .env.example .env
```

Edit `.env` as needed. Key variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | `8080` | Backend listen port |
| `DB_PATH` | `data/autocheck.db` | SQLite file path |
| `IMPORT_DIR` | `data/imports` | Watched directory for CSV files |
| `IMPORT_CRON` | _(empty)_ | Cron expression for scheduled imports |
| `IMPORT_FILE` | _(empty)_ | Fixed CSV path for scheduled imports |
| `IMPORT_ENABLED` | `true` | Enable/disable scheduler |
| `IMPORT_TIMEOUT` | `5m` | Max duration per import run |

#### 3. Run the server

```sh
go run ./cmd/server
```

The server starts on `http://localhost:8080` (or the port set in `HTTP_PORT`).

#### 4. Run backend tests

```sh
go test ./...
```

---

### Frontend (React + Vite)

The frontend lives in [`web/`](web/) and proxies `/api` requests to the Go backend during development.

#### 1. Install dependencies

```sh
cd web
npm install
```

#### 2. Start the dev server

```sh
npm run dev
```

The frontend will be available at `http://localhost:5173` and proxies API calls to `http://localhost:8080`.

> **Important:** Make sure the Go backend is running on port 8080 before starting the frontend dev server.

#### 3. Build for production

```sh
cd web
npm run build
```

Output goes to `web/dist/`. The Docker multi-stage build handles this automatically.

---

## Configuration

All configuration is via environment variables. See [`.env.example`](.env.example) for the full list.

### Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `HTTP_PORT` | `8080` | No | HTTP listen port |
| `DB_PATH` | `data/autocheck.db` | No | SQLite database file path |
| `IMPORT_DIR` | `data/imports` | No | Directory watched for 1C export files |
| `IMPORT_CRON` | _(empty)_ | No | 5-field cron expression (e.g. `0 * * * *`) |
| `IMPORT_FILE` | _(empty)_ | No | Fixed CSV path imported on each scheduled run |
| `IMPORT_ENABLED` | `true` | No | Enable or disable the scheduled importer |
| `IMPORT_TIMEOUT` | `5m` | No | Max duration per scheduled run before failure |

### Scheduled Imports

To enable scheduled imports, set both `IMPORT_CRON` and `IMPORT_FILE`:

```env
IMPORT_CRON=0 * * * *    # every hour at minute 0
IMPORT_FILE=data/imports/cars.csv
```

If either is missing, the scheduler logs a warning and does not start.

---

## Running Tests

### Backend tests

```sh
go test ./...
```

### Frontend tests

```sh
cd web
npm test
```

---

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Liveness probe |
| `GET` | `/api/cars?page=&limit=` | Paginated car list |
| `GET` | `/api/cars/{vin}` | Single car by VIN |
| `POST` | `/api/import` | Manual import trigger (multipart/form-data, field `file`) |
| `GET` | `/api/import/status` | Last import run summary |

### Example: Manual import via curl

```sh
curl -X POST http://localhost:8080/api/import \
  -F "file=@data/imports/partner_1c_export.csv"
```

### Example: Check import status

```sh
curl http://localhost:8080/api/import/status
```

---

## Frontend Development

The React frontend is built with Vite.

| Command | Description |
|---------|-------------|
| `npm run dev` | Start dev server with hot reload (proxies `/api` to Go backend) |
| `npm run build` | Production build into `web/dist/` |
| `npm run preview` | Preview production build locally |

### Dev server proxy

The Vite dev server proxies `/api/*` to `http://localhost:8080` (see [`web/vite.config.js`](web/vite.config.js)). This means you can run the frontend and backend independently during development.

---

## Project Structure

```
cmd/
  server/              # Entrypoint: config, database, router, scheduler
  gen-dataset/         # Utility to generate test CSV datasets
internal/
  api/                 # HTTP handlers and router
  config/              # Environment-based configuration loader
  database/            # SQLite connection, WAL mode, migrations
  domain/              # Domain models (Car, ImportRun, ImportError)
  importer/            # File loading, parsing, normalization, upsert orchestration
  parser/              # 1C CSV parsing (encoding, delimiters, column mapping)
  repository/          # SQLite data access (cars table, upsert, queries)
  scheduler/           # Cron-based scheduled import with timeout
web/                   # React frontend (Vite)
migrations/            # SQL schema files
data/imports/          # Watched directory for 1C export files
scripts/               # Python utilities (dataset generation)
```

---

## Troubleshooting

### Port already in use

If `8080` is taken, set a different port:

```sh
# .env
HTTP_PORT=3000
```

Or with Docker Compose:

```sh
HTTP_PORT=3000 docker compose up
```

### Database locked / SQLite errors

SQLite allows one writer at a time. The service sets `max_open_conns=1` to avoid lock contention. If you see lock errors:

1. Ensure no other process is writing to the same `DB_PATH`.
2. Check that the `data/` directory is writable.

### Import fails with "no such file"

Make sure `IMPORT_FILE` points to an existing file and `IMPORT_DIR` is correctly set. For Docker, paths are inside the container:

```yaml
# docker-compose.yml
IMPORT_FILE=/app/data/imports/cars.csv
```

### Frontend shows blank page / API errors

1. Verify the backend is running: `curl http://localhost:8080/api/health`
2. Check browser console for CORS or network errors.
3. Ensure `vite.config.js` proxy target matches the backend port.

### Scheduler not starting

The scheduler requires both `IMPORT_CRON` and `IMPORT_FILE` to be set. Check server logs:

```
scheduler: not configured (IMPORT_CRON or IMPORT_FILE missing)
```

---

## Status

Project skeleton with implemented business logic. See [AI_LOGS.md](AI_LOGS.md) for progress and [AI_RESPONSES/](AI_RESPONSES/) for design documents.
