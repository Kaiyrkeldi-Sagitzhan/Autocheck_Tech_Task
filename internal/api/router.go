// Package api contains HTTP handlers and routing.
package api

import (
	"encoding/json"
	"net/http"
)

// NewRouter builds the HTTP mux with all routes registered.
// Handlers are stubs for now; business logic is wired in later phases.
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealth)

	mux.HandleFunc("GET /api/v1/cars", notImplemented)
	mux.HandleFunc("GET /api/v1/cars/{vin}", notImplemented)
	mux.HandleFunc("POST /api/v1/import", notImplemented)
	mux.HandleFunc("GET /api/v1/import/status", notImplemented)
	mux.HandleFunc("GET /api/v1/import/runs/{id}/errors", notImplemented)

	return mux
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented yet"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
