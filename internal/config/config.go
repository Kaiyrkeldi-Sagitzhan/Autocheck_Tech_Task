// Package config loads service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

// Config holds all runtime configuration for the service.
type Config struct {
	HTTPPort       string        // HTTP_PORT, default "8080"
	DBPath         string        // DB_PATH, default "data/autocheck.db"
	ImportDir      string        // IMPORT_DIR, default "data/imports"
	ImportCron     string        // IMPORT_CRON, default empty (scheduler disabled)
	ImportFile     string        // IMPORT_FILE, default empty
	ImportEnabled  bool          // IMPORT_ENABLED, default true
	ImportTimeout  time.Duration // IMPORT_TIMEOUT, default 5m
}

// Load reads configuration from the environment, applying defaults.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:  envOr("HTTP_PORT", "8080"),
		DBPath:    envOr("DB_PATH", "data/autocheck.db"),
		ImportDir: envOr("IMPORT_DIR", "data/imports"),
		ImportCron: envOr("IMPORT_CRON", ""),
		ImportFile: envOr("IMPORT_FILE", ""),
	}

	// IMPORT_ENABLED
	cfg.ImportEnabled = true
	if v := os.Getenv("IMPORT_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid IMPORT_ENABLED %q: %w", v, err)
		}
		cfg.ImportEnabled = b
	}

	// IMPORT_TIMEOUT
	timeout := envOr("IMPORT_TIMEOUT", "5m")
	d, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid IMPORT_TIMEOUT %q: %w", timeout, err)
	}
	cfg.ImportTimeout = d

	// Validate IMPORT_CRON if set
	if cfg.ImportCron != "" {
		_, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(cfg.ImportCron)
		if err != nil {
			return nil, fmt.Errorf("invalid IMPORT_CRON %q: %w", cfg.ImportCron, err)
		}
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
