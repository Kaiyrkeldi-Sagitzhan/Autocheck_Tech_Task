package database

// schemaSQL is the embedded schema, kept in sync with migrations/001_schema.sql.
// It is embedded here so the binary is self-contained; the migrations/ directory
// remains the human-readable source of truth.
const schemaSQL = `
CREATE TABLE IF NOT EXISTS cars (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    vin           TEXT NOT NULL UNIQUE,
    brand         TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    year          INTEGER,
    mileage_km    INTEGER,
    price         INTEGER,
    currency      TEXT NOT NULL DEFAULT 'KZT',
    color         TEXT,
    engine        TEXT,
    transmission  TEXT,
    body_type     TEXT,
    defects       TEXT NOT NULL DEFAULT '[]',
    defects_raw   TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active',
    source_file   TEXT NOT NULL DEFAULT '',
    imported_at   DATETIME NOT NULL,
    updated_at    DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cars_brand_model ON cars(brand, model);
CREATE INDEX IF NOT EXISTS idx_cars_status      ON cars(status);

CREATE TABLE IF NOT EXISTS import_runs (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger_type  TEXT NOT NULL,
    file_name     TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL,
    rows_total    INTEGER NOT NULL DEFAULT 0,
    created       INTEGER NOT NULL DEFAULT 0,
    updated       INTEGER NOT NULL DEFAULT 0,
    skipped       INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    started_at    DATETIME NOT NULL,
    finished_at   DATETIME
);

CREATE TABLE IF NOT EXISTS import_errors (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id     INTEGER NOT NULL REFERENCES import_runs(id),
    row_number INTEGER NOT NULL,
    reason     TEXT NOT NULL,
    raw_row    TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_import_errors_run ON import_errors(run_id);
`
