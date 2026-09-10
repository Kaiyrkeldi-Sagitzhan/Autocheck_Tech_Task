// Package config loads service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration for the service.
type Config struct {
	HTTPPort       string        // HTTP_PORT, default "8080"
	DBPath         string        // DB_PATH, default "data/autocheck.db"
	ImportDir      string        // IMPORT_DIR, default "data/imports"
	ImportInterval time.Duration // IMPORT_INTERVAL, default 5m
	ImportCron     string        // IMPORT_CRON, default empty (scheduler disabled)
	ImportFile     string        // IMPORT_FILE, default empty
}

// Load reads configuration from the environment, applying defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:   envOr("HTTP_PORT", "8080"),
		DBPath:     envOr("DB_PATH", "data/autocheck.db"),
		ImportDir:  envOr("IMPORT_DIR", "data/imports"),
		ImportCron: envOr("IMPORT_CRON", ""),
		ImportFile: envOr("IMPORT_FILE", ""),
	}

	interval := envOr("IMPORT_INTERVAL", "5m")
	d, err := time.ParseDuration(interval)
	if err != nil {
		return nil, fmt.Errorf("invalid IMPORT_INTERVAL %q: %w", interval, err)
	}
	cfg.ImportInterval = d

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
