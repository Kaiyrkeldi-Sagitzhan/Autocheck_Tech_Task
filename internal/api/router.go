// Package api contains HTTP handlers and routing.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"awesomeProject5/internal/importer"
	"awesomeProject5/internal/repository"
)

// NewRouter builds the HTTP mux with all routes registered.
// Dependencies are injected to keep handlers thin and testable.
func NewRouter(repo *repository.Repository, imp *importer.Importer) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/cars", handleListCars(repo))
	mux.HandleFunc("GET /api/cars/{vin}", handleGetCarByVIN(repo))
	mux.HandleFunc("POST /api/import", handleImport(imp))

	// Wrap with CORS and JSON content-type middleware.
	return corsMiddleware(jsonMiddleware(mux))
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleListCars(repo *repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

		cars, total, err := repo.ListCars(r.Context(), page, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"items":     cars,
			"total":     total,
			"page":      page,
			"page_size": limit,
		})
	}
}

func handleGetCarByVIN(repo *repository.Repository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vin := r.PathValue("vin")
		if vin == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "vin is required"})
			return
		}

		car, err := repo.CarByVIN(r.Context(), vin)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		if car == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "car not found"})
			return
		}

		writeJSON(w, http.StatusOK, car)
	}
}

func handleImport(imp *importer.Importer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
			return
		}
		defer file.Close()

		buf := make([]byte, header.Size)
		if _, err := file.Read(buf); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read file"})
			return
		}

		report, err := imp.Import(r.Context(), header.Filename, buf)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "import failed: " + err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, report)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip JSON content-type check for health and import endpoints
		if r.URL.Path == "/api/health" || r.URL.Path == "/api/import" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Header.Get("Content-Type") != "" && r.Header.Get("Content-Type") != "application/json" {
			writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "unsupported media type"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
