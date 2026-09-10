package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"awesomeProject5/internal/database"
	"awesomeProject5/internal/importer"
	"awesomeProject5/internal/repository"
)

func setupTestServer(t *testing.T) (*repository.Repository, *importer.Importer, http.Handler) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := repository.New(db)
	imp := importer.New(repo)
	handler := NewRouter(repo, imp)
	return repo, imp, handler
}

func TestHealth(t *testing.T) {
	_, _, handler := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Errorf("expected status ok in body, got %s", w.Body.String())
	}
}

func TestListCars(t *testing.T) {
	_, _, handler := setupTestServer(t)

	// Insert a car first.
	req := httptest.NewRequest("GET", "/api/cars?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := decodeJSON(w.Body, &resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if resp["total"] != float64(0) {
		t.Errorf("expected total 0, got %v", resp["total"])
	}
	if resp["page"] != float64(1) {
		t.Errorf("expected page 1, got %v", resp["page"])
	}
	if resp["page_size"] != float64(10) {
		t.Errorf("expected page_size 10, got %v", resp["page_size"])
	}
}

func TestGetCarByVIN(t *testing.T) {
	_, _, handler := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/cars/NONEXISTENT", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "car not found") {
		t.Errorf("expected 'car not found' in body, got %s", w.Body.String())
	}
}

func TestImportFile(t *testing.T) {
	_, _, handler := setupTestServer(t)

	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	body := &bytes.Buffer{}
	body.WriteString("------WebKitFormBoundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.csv\"\r\n")
	body.WriteString("Content-Type: text/csv\r\n\r\n")
	body.Write([]byte(csvData))
	body.WriteString("\r\n------WebKitFormBoundary--\r\n")

	req := httptest.NewRequest("POST", "/api/import", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundary")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := decodeJSON(w.Body, &resp); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if resp["created"] != float64(1) {
		t.Errorf("expected created 1, got %v", resp["created"])
	}
	if resp["rows_total"] != float64(1) {
		t.Errorf("expected rows_total 1, got %v", resp["rows_total"])
	}
}

func TestImportInvalidUpload(t *testing.T) {
	_, _, handler := setupTestServer(t)

	// No file field.
	body := &bytes.Buffer{}
	body.WriteString("------WebKitFormBoundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"other\"\r\n\r\n")
	body.WriteString("value\r\n")
	body.WriteString("------WebKitFormBoundary--\r\n")

	req := httptest.NewRequest("POST", "/api/import", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundary")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func decodeJSON(body *bytes.Buffer, v any) error {
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// repoCar is a minimal struct for test setup (not used directly in assertions).
type repoCar struct {
	VIN string
}
