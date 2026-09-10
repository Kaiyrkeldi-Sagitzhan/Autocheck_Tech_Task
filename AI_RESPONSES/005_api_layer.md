# Task 005: REST API Layer

## Overview
Implemented the HTTP layer for the Autocheck.kz car inventory import microservice. The API exposes endpoints for health checks, car listing, car lookup by VIN, and CSV file import.

## Files Created
- `internal/api/router_test.go` — API handler tests (5 test functions)

## Files Modified
- `internal/domain/car.go` — Added JSON struct tags to `Car` for HTTP serialization
- `internal/repository/repository.go` — Added `ListCars` with pagination and `CarByVIN`
- `internal/api/router.go` — Implemented handlers, middleware, and dependency injection
- `cmd/server/main.go` — Wired repository and importer dependencies into the router

## Architecture Decisions

### Dependency Injection
Dependencies are injected via constructor parameters rather than using global state:
```go
func NewRouter(repo *repository.Repository, imp *importer.Importer) http.Handler
```

### Middleware Stack
1. **CORS** — Allows all origins, methods GET/POST/OPTIONS, headers Content-Type
2. **JSON Content-Type** — Enforces `application/json` for JSON endpoints; exempts `/api/health` and `/api/import` (multipart form)
3. **Logging** — Logs method, path, and remote address

### Endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Health check |
| GET | `/api/cars` | Paginated car listing (`?page=1&limit=20`) |
| GET | `/api/cars/{vin}` | Single car by VIN (404 if not found) |
| POST | `/api/import` | Multipart CSV file import |

### Pagination
- Defaults: `page=1`, `limit=20`
- Maximum: `limit=100`
- Response includes `items`, `total`, `page`, `page_size`

### Error Handling
- 400 for bad requests (missing VIN, invalid multipart)
- 404 for car not found
- 415 for unsupported media type on JSON endpoints
- 500 for internal errors

## Test Results
All 24 tests pass:
- `internal/api` — 5 tests
- `internal/importer` — 6 tests
- `internal/parser` — 13 tests
- `internal/repository` — 5 tests

## Docker Verification
- Image builds successfully
- Container starts and listens on `:8080`
- All endpoints respond correctly via curl
