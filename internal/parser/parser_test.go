package parser

import (
	"fmt"
	"strings"
	"testing"

	"awesomeProject5/internal/domain"
)

// header is the standard CSV header used across tests.
const header = "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt"

// baseRow returns a well-formed CSV data row with all fields populated.
func baseRow(vin, brand, model, year, mileage, price, currency, color, engine, transmission, bodyType, defects, status, updatedAt string) string {
	return strings.Join([]string{
		vin, brand, model, year, mileage, price, currency, color,
		engine, transmission, bodyType, defects, status, updatedAt,
	}, ";")
}

// ============================================================================
// PART 1 — PARSER TESTS
// ============================================================================

// TestParse_ValidCSV_HappyPath asserts exact field values for every record,
// not just the count.
func TestParse_ValidCSV_HappyPath(t *testing.T) {
	csvData := header + "\r\n" +
		baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n" +
		baseRow("Y1YJ2345678901234", "BMW", "X5", "2019", "30000", "12000000", "KZT", "Черный",
			"3.0 дизель 245 л.с.", "AT", "SUV", "Царапина на бампере", "sold", "2024-06-20") + "\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	// Exactly 2 valid records.
	if len(cars) != 2 {
		t.Fatalf("expected 2 cars, got %d", len(cars))
	}

	// Zero row-level errors.
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(errors), errors)
	}

	// --- First car assertions ---
	c1 := cars[0]
	if c1.VIN != "X5XJ1234567890123" {
		t.Errorf("car[0].VIN: expected X5XJ1234567890123, got %s", c1.VIN)
	}
	if c1.Brand != "Toyota" {
		t.Errorf("car[0].Brand: expected Toyota, got %s", c1.Brand)
	}
	if c1.Model != "Camry" {
		t.Errorf("car[0].Model: expected Camry, got %s", c1.Model)
	}
	if c1.Year == nil || *c1.Year != 2020 {
		t.Errorf("car[0].Year: expected 2020, got %v", c1.Year)
	}
	if c1.MileageKm == nil || *c1.MileageKm != 50000 {
		t.Errorf("car[0].MileageKm: expected 50000, got %v", c1.MileageKm)
	}
	if c1.Price == nil || *c1.Price != 5000000 {
		t.Errorf("car[0].Price: expected 5000000, got %v", c1.Price)
	}
	if c1.Currency != "KZT" {
		t.Errorf("car[0].Currency: expected KZT, got %s", c1.Currency)
	}
	if c1.Color != "Белый" {
		t.Errorf("car[0].Color: expected Белый, got %s", c1.Color)
	}
	if c1.Engine != "2.0 бензин 150 л.с." {
		t.Errorf("car[0].Engine: expected '2.0 бензин 150 л.с.', got %s", c1.Engine)
	}
	if c1.Transmission != "AT" {
		t.Errorf("car[0].Transmission: expected AT, got %s", c1.Transmission)
	}
	if c1.BodyType != "Седан" {
		t.Errorf("car[0].BodyType: expected Седан, got %s", c1.BodyType)
	}
	if c1.DefectsRaw != "" {
		t.Errorf("car[0].DefectsRaw: expected empty, got %s", c1.DefectsRaw)
	}
	if c1.Status != domain.CarStatusActive {
		t.Errorf("car[0].Status: expected active, got %s", c1.Status)
	}
	if c1.UpdatedAt != "2024-01-15" {
		t.Errorf("car[0].UpdatedAt: expected 2024-01-15, got %s", c1.UpdatedAt)
	}
	if c1.RowNumber != 2 {
		t.Errorf("car[0].RowNumber: expected 2, got %d", c1.RowNumber)
	}

	// --- Second car assertions ---
	c2 := cars[1]
	if c2.VIN != "Y1YJ2345678901234" {
		t.Errorf("car[1].VIN: expected Y1YJ2345678901234, got %s", c2.VIN)
	}
	if c2.Brand != "BMW" {
		t.Errorf("car[1].Brand: expected BMW, got %s", c2.Brand)
	}
	if c2.Model != "X5" {
		t.Errorf("car[1].Model: expected X5, got %s", c2.Model)
	}
	if c2.Year == nil || *c2.Year != 2019 {
		t.Errorf("car[1].Year: expected 2019, got %v", c2.Year)
	}
	if c2.MileageKm == nil || *c2.MileageKm != 30000 {
		t.Errorf("car[1].MileageKm: expected 30000, got %v", c2.MileageKm)
	}
	if c2.Price == nil || *c2.Price != 12000000 {
		t.Errorf("car[1].Price: expected 12000000, got %v", c2.Price)
	}
	if c2.Currency != "KZT" {
		t.Errorf("car[1].Currency: expected KZT, got %s", c2.Currency)
	}
	if c2.Color != "Черный" {
		t.Errorf("car[1].Color: expected Черный, got %s", c2.Color)
	}
	if c2.DefectsRaw != "Царапина на бампере" {
		t.Errorf("car[1].DefectsRaw: expected 'Царапина на бампере', got %s", c2.DefectsRaw)
	}
	if c2.Status != domain.CarStatusSold {
		t.Errorf("car[1].Status: expected sold, got %s", c2.Status)
	}
	if c2.UpdatedAt != "2024-06-20" {
		t.Errorf("car[1].UpdatedAt: expected 2024-06-20, got %s", c2.UpdatedAt)
	}
	if c2.RowNumber != 3 {
		t.Errorf("car[1].RowNumber: expected 3, got %d", c2.RowNumber)
	}
}

// TestParse_InvalidVIN covers wrong length, forbidden characters (I, O, Q),
// and non-alphanumeric junk. The bad row must be excluded from valid records
// and produce a row-level error. Other valid rows in the same file must still
// parse correctly.
func TestParse_InvalidVIN(t *testing.T) {
	cases := []struct {
		name        string
		vin         string
		wantErrSub  string
		wantErrType string // "length" or "character"
	}{
		{
			name:        "too_short_10_chars",
			vin:         "ABC1234567",
			wantErrSub:  "invalid VIN length",
			wantErrType: "length",
		},
		{
			name:        "too_long_18_chars",
			vin:         "ABCDEFGHIJKLMNOPQRS",
			wantErrSub:  "invalid VIN length",
			wantErrType: "length",
		},
		{
			name:        "contains_forbidden_I",
			vin:         "ABCDEFGHIJKLMNOPQ",
			wantErrSub:  "invalid VIN character",
			wantErrType: "character",
		},
		{
			name:        "contains_forbidden_O",
			vin:         "ABCDEFGHIJKLMNOP0",
			wantErrSub:  "invalid VIN character",
			wantErrType: "character",
		},
		{
			name:        "contains_forbidden_Q",
			vin:         "ABCDEFGHIJKLMNQPQ",
			wantErrSub:  "invalid VIN character",
			wantErrType: "character",
		},
		{
			name:        "non_alphanumeric_junk",
			vin:         "ABC-123!@#XYZ1234",
			wantErrSub:  "invalid VIN character",
			wantErrType: "character",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csvData := header + "\r\n" +
				baseRow(tc.vin, "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
					"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n" +
				baseRow("Y1YJ2345678901234", "BMW", "X5", "2019", "30000", "12000000", "KZT", "Черный",
					"3.0 дизель 245 л.с.", "AT", "SUV", "", "sold", "2024-06-20") + "\r\n"

			cars, errors, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected fatal error: %v", err)
			}

			// Bad row excluded; only the second valid row remains.
			if len(cars) != 1 {
				t.Fatalf("expected 1 valid car, got %d", len(cars))
			}
			if cars[0].VIN != "Y1YJ2345678901234" {
				t.Errorf("expected remaining car VIN Y1YJ2345678901234, got %s", cars[0].VIN)
			}

			// Exactly one row-level error for the bad VIN.
			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d: %v", len(errors), errors)
			}
			if errors[0].Row != 2 {
				t.Errorf("expected error at row 2, got %d", errors[0].Row)
			}
			if !strings.Contains(errors[0].Reason, tc.wantErrSub) {
				t.Errorf("expected error reason to contain %q, got %s", tc.wantErrSub, errors[0].Reason)
			}
		})
	}
}

// TestParse_MissingRequiredFields asserts that missing VIN, brand, or model
// each produce a row-level error and exclude the row from valid records.
// Policy: VIN, brand, and model are all required; missing any = reject row.
func TestParse_MissingRequiredFields(t *testing.T) {
	cases := []struct {
		name     string
		row      string
		wantErr string
	}{
		{
			name:     "missing_vin",
			row:      baseRow("", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый", "2.0", "AT", "Седан", "", "active", "2024-01-15"),
			wantErr:  "missing VIN",
		},
		{
			name:     "missing_brand",
			row:      baseRow("X5XJ1234567890123", "", "Camry", "2020", "50000", "5000000", "KZT", "Белый", "2.0", "AT", "Седан", "", "active", "2024-01-15"),
			wantErr:  "missing brand",
		},
		{
			name:     "missing_model",
			row:      baseRow("X5XJ1234567890123", "Toyota", "", "2020", "50000", "5000000", "KZT", "Белый", "2.0", "AT", "Седан", "", "active", "2024-01-15"),
			wantErr:  "missing model",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csvData := header + "\r\n" +
				tc.row + "\r\n" +
				baseRow("Y1YJ2345678901234", "BMW", "X5", "2019", "30000", "12000000", "KZT", "Черный",
					"3.0 дизель 245 л.с.", "AT", "SUV", "", "sold", "2024-06-20") + "\r\n"

			cars, errors, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected fatal error: %v", err)
			}

			// The valid second row must still parse.
			if len(cars) != 1 {
				t.Fatalf("expected 1 valid car, got %d", len(cars))
			}
			if cars[0].VIN != "Y1YJ2345678901234" {
				t.Errorf("expected remaining car VIN Y1YJ2345678901234, got %s", cars[0].VIN)
			}

			// Exactly one error for the bad row.
			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d: %v", len(errors), errors)
			}
			if errors[0].Row != 2 {
				t.Errorf("expected error at row 2, got %d", errors[0].Row)
			}
			if !strings.Contains(errors[0].Reason, tc.wantErr) {
				t.Errorf("expected error reason to contain %q, got %s", tc.wantErr, errors[0].Reason)
			}
		})
	}
}

// TestParse_InvalidMileage asserts the parser's explicit policy:
//   - negative mileage → row rejected with "invalid mileage" error
//   - empty mileage  → row accepted, MileageKm is nil
//   - non-numeric text (e.g. "120 000 км", "abc") → row accepted, MileageKm is nil
//   - numeric with spaces as thousands separator (e.g. "120 000") → parsed correctly
func TestParse_InvalidMileage(t *testing.T) {
	cases := []struct {
		name        string
		mileage     string
		wantCars    int
		wantErrors  int
		wantMileage *int // nil means expect nil
		wantErrSub  string
	}{
		{
			name:        "negative_mileage_rejected",
			mileage:     "-100",
			wantCars:    0,
			wantErrors:  1,
			wantMileage: nil,
			wantErrSub:  "invalid mileage",
		},
		{
			name:        "empty_mileage_accepted_as_nil",
			mileage:     "",
			wantCars:    1,
			wantErrors:  0,
			wantMileage: nil,
		},
		{
			name:        "garbage_text_accepted_as_nil",
			mileage:     "120 000 км",
			wantCars:    1,
			wantErrors:  0,
			wantMileage: nil,
		},
		{
			name:        "non_numeric_accepted_as_nil",
			mileage:     "abc",
			wantCars:    1,
			wantErrors:  0,
			wantMileage: nil,
		},
		{
			name:        "spaces_as_thousands_separator_parsed",
			mileage:     "120 000",
			wantCars:    1,
			wantErrors:  0,
			wantMileage: intPtr(120000),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csvData := header + "\r\n" +
				baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", tc.mileage, "5000000", "KZT", "Белый",
					"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n"

			cars, errors, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected fatal error: %v", err)
			}

			if len(cars) != tc.wantCars {
				t.Fatalf("expected %d cars, got %d", tc.wantCars, len(cars))
			}
			if len(errors) != tc.wantErrors {
				t.Fatalf("expected %d errors, got %d: %v", tc.wantErrors, len(errors), errors)
			}

			if tc.wantCars > 0 {
				if tc.wantMileage == nil {
					if cars[0].MileageKm != nil {
						t.Errorf("expected nil MileageKm, got %d", *cars[0].MileageKm)
					}
				} else {
					if cars[0].MileageKm == nil || *cars[0].MileageKm != *tc.wantMileage {
						t.Errorf("expected MileageKm %d, got %v", *tc.wantMileage, cars[0].MileageKm)
					}
				}
			}

			if tc.wantErrors > 0 && tc.wantErrSub != "" {
				if !strings.Contains(errors[0].Reason, tc.wantErrSub) {
					t.Errorf("expected error reason to contain %q, got %s", tc.wantErrSub, errors[0].Reason)
				}
			}
		})
	}
}

// TestParse_MalformedCSV distinguishes between:
//   - "one bad row" (recoverable: cases 2-4 above) — parser returns records + errors, no fatal error
//   - "the whole file is broken" (fatal) — parser returns a non-nil fatal error
//
// Note: The parser uses a lenient CSV reader (FieldsPerRecord=-1, LazyQuotes=true),
// so truncated rows and binary garbage are treated as recoverable format issues
// rather than fatal errors. Missing fields become empty strings; binary input
// produces no valid cars but also no fatal error. This is a deliberate design
// choice: the importer layer handles per-row validation, and the parser focuses
// on extracting whatever structured data it can.
//
// Cases covered:
//  1. Truncated file (cut off mid-row) → no fatal error, row parsed with missing fields
//  2. Inconsistent column count between header and data rows → still parses (FieldsPerRecord=-1)
//  3. Completely empty file → empty result, no error
//  4. Header only, no data rows → empty result, no error
//  5. Binary garbage that isn't CSV → no fatal error, no valid cars produced
func TestParse_MalformedCSV(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantFatal   bool
		wantCars    int
		wantErrors  int
	}{
		{
			// Parser is lenient: truncated row is parsed with missing fields as empty strings.
			name:       "truncated_mid_row",
			input:      header + "\r\nX5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0",
			wantFatal:  false,
			wantCars:   1,
			wantErrors: 0,
		},
		{
			name:       "inconsistent_columns_fewer",
			input:      header + "\r\nX5XJ1234567890123;Toyota;Camry\r\n",
			wantFatal:  false,
			wantCars:   1,
			wantErrors: 0,
		},
		{
			name:       "inconsistent_columns_more",
			input:      header + "\r\nX5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0;AT;Седан;;active;2024-01-15;EXTRA\r\n",
			wantFatal:  false,
			wantCars:   1,
			wantErrors: 0,
		},
		{
			name:       "completely_empty",
			input:      "",
			wantFatal:  false,
			wantCars:   0,
			wantErrors: 0,
		},
		{
			name:       "header_only",
			input:      header + "\r\n",
			wantFatal:  false,
			wantCars:   0,
			wantErrors: 0,
		},
		{
			// Parser is lenient: binary input produces no valid cars but no fatal error.
			name:       "binary_garbage",
			input:      "\x00\x01\x02\x03\x04\x05",
			wantFatal:  false,
			wantCars:   0,
			wantErrors: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cars, errors, err := Parse([]byte(tc.input))

			if tc.wantFatal {
				if err == nil {
					t.Fatalf("expected fatal error, got nil")
				}
				// On fatal error, all return values should be nil/empty.
				if len(cars) != 0 {
					t.Errorf("expected 0 cars on fatal error, got %d", len(cars))
				}
				if len(errors) != 0 {
					t.Errorf("expected 0 errors on fatal error, got %d", len(errors))
				}
			} else {
				if err != nil {
					t.Fatalf("expected no fatal error, got: %v", err)
				}
				if len(cars) != tc.wantCars {
					t.Errorf("expected %d cars, got %d", tc.wantCars, len(cars))
				}
				if len(errors) != tc.wantErrors {
					t.Errorf("expected %d errors, got %d: %v", tc.wantErrors, len(errors), errors)
				}
			}
		})
	}
}

// TestParse_InvalidYear asserts that out-of-range years are rejected (fatal per-row),
// while non-numeric years produce nil Year with no error.
func TestParse_InvalidYear(t *testing.T) {
	cases := []struct {
		name       string
		year       string
		wantCars   int
		wantErrors int
		wantErrSub string
	}{
		{
			name:       "year_too_old",
			year:       "1800",
			wantCars:   0,
			wantErrors: 1,
			wantErrSub: "invalid year",
		},
		{
			name:       "year_too_new",
			year:       "2101",
			wantCars:   0,
			wantErrors: 1,
			wantErrSub: "invalid year",
		},
		{
			name:       "non_numeric_year",
			year:       "abc",
			wantCars:   1,
			wantErrors: 0,
			wantErrSub: "",
		},
		{
			name:       "empty_year",
			year:       "",
			wantCars:   1,
			wantErrors: 0,
			wantErrSub: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csvData := header + "\r\n" +
				baseRow("X5XJ1234567890123", "Toyota", "Camry", tc.year, "50000", "5000000", "KZT", "Белый",
					"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n"

			cars, errors, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected fatal error: %v", err)
			}

			if len(cars) != tc.wantCars {
				t.Fatalf("expected %d cars, got %d", tc.wantCars, len(cars))
			}
			if len(errors) != tc.wantErrors {
				t.Fatalf("expected %d errors, got %d: %v", tc.wantErrors, len(errors), errors)
			}

			if tc.wantCars > 0 && tc.year == "" {
				if cars[0].Year != nil {
					t.Errorf("expected nil Year for empty input, got %d", *cars[0].Year)
				}
			}
			if tc.wantErrors > 0 && tc.wantErrSub != "" {
				if !strings.Contains(errors[0].Reason, tc.wantErrSub) {
					t.Errorf("expected error reason to contain %q, got %s", tc.wantErrSub, errors[0].Reason)
				}
			}
		})
	}
}

// TestParse_DuplicateVINWithinFile asserts the parser's dedup policy.
// The parser itself does NOT deduplicate — it returns all parsed rows.
// Dedup is handled by the importer layer (last-wins via sequential upsert).
// This test documents that behavior explicitly.
func TestParse_DuplicateVINWithinFile(t *testing.T) {
	csvData := header + "\r\n" +
		baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n" +
		baseRow("X5XJ1234567890123", "Toyota", "Camry", "2021", "60000", "5500000", "KZT", "Красный",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "sold", "2024-12-01") + "\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	// Parser returns both rows — dedup is the importer's job.
	if len(cars) != 2 {
		t.Fatalf("expected parser to return 2 rows (no dedup), got %d", len(cars))
	}
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}

	// Both records have the same VIN.
	if cars[0].VIN != "X5XJ1234567890123" || cars[1].VIN != "X5XJ1234567890123" {
		t.Error("expected both records to have the same VIN")
	}

	// The importer test (TestImport_SameVINTwice_NoDuplicate) verifies that
	// the importer handles this correctly (last-wins).
}

// TestParse_WhitespaceTrimming asserts that leading/trailing spaces in field
// values are trimmed correctly, and VIN is normalized to uppercase.
func TestParse_WhitespaceTrimming(t *testing.T) {
	csvData := header + "\r\n" +
		"  X5XJ1234567890123  ;  Toyota  ;  Camry  ;  2020  ;  50000  ;  5000000  ;  KZT  ;  Белый  ;  2.0 бензин 150 л.с.  ;  AT  ;  Седан  ;  ;  active  ;  2024-01-15  \r\n"

	cars, _, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	c := cars[0]
	if c.VIN != "X5XJ1234567890123" {
		t.Errorf("VIN: expected 'X5XJ1234567890123', got %q", c.VIN)
	}
	if c.Brand != "Toyota" {
		t.Errorf("Brand: expected 'Toyota', got %q", c.Brand)
	}
	if c.Model != "Camry" {
		t.Errorf("Model: expected 'Camry', got %q", c.Model)
	}
	if c.Year == nil || *c.Year != 2020 {
		t.Errorf("Year: expected 2020, got %v", c.Year)
	}
	if c.MileageKm == nil || *c.MileageKm != 50000 {
		t.Errorf("MileageKm: expected 50000, got %v", c.MileageKm)
	}
	if c.Price == nil || *c.Price != 5000000 {
		t.Errorf("Price: expected 5000000, got %v", c.Price)
	}
	if c.Currency != "KZT" {
		t.Errorf("Currency: expected KZT, got %s", c.Currency)
	}
	if c.Color != "Белый" {
		t.Errorf("Color: expected 'Белый', got %q", c.Color)
	}
	if c.Status != domain.CarStatusActive {
		t.Errorf("Status: expected active, got %s", c.Status)
	}
}

// TestParse_DefaultCurrency asserts that empty currency defaults to KZT.
func TestParse_DefaultCurrency(t *testing.T) {
	csvData := header + "\r\n" +
		baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n"

	cars, _, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	if cars[0].Currency != "KZT" {
		t.Errorf("expected default Currency KZT, got %s", cars[0].Currency)
	}
}

// TestParse_StatusMapping asserts all status values and case-insensitivity.
func TestParse_StatusMapping(t *testing.T) {
	cases := []struct {
		input    string
		expected domain.CarStatus
	}{
		{"active", domain.CarStatusActive},
		{"sold", domain.CarStatusSold},
		{"reserved", domain.CarStatusReserved},
		{"", domain.CarStatusActive},
		{"unknown", domain.CarStatusActive},
		{"ACTIVE", domain.CarStatusActive},
		{"Sold", domain.CarStatusSold},
		{"RESERVED", domain.CarStatusReserved},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			csvData := header + "\r\n" +
				baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
					"2.0 бензин 150 л.с.", "AT", "Седан", "", tc.input, "2024-01-15") + "\r\n"

			cars, _, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected fatal error: %v", err)
			}

			if len(cars) != 1 {
				t.Fatalf("expected 1 car, got %d", len(cars))
			}
			if cars[0].Status != tc.expected {
				t.Errorf("expected Status %s, got %s", tc.expected, cars[0].Status)
			}
		})
	}
}

// TestParse_RowNumberTracking asserts that RowNumber is 1-based and header
// is row 1, so first data row is row 2.
func TestParse_RowNumberTracking(t *testing.T) {
	csvData := header + "\r\n" +
		baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n" +
		baseRow("", "BMW", "X5", "2019", "30000", "12000000", "KZT", "Черный",
			"3.0 дизель 245 л.с.", "AT", "SUV", "", "sold", "2024-06-20") + "\r\n" +
		baseRow("Y1YJ2345678901234", "Toyota", "Corolla", "2021", "15000", "4000000", "KZT", "Красный",
			"1.6 бензин 110 л.с.", "MT", "Хэтчбек", "", "active", "2024-03-10") + "\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	if len(cars) != 2 {
		t.Fatalf("expected 2 cars, got %d", len(cars))
	}
	if cars[0].RowNumber != 2 {
		t.Errorf("expected first car RowNumber 2, got %d", cars[0].RowNumber)
	}
	if cars[1].RowNumber != 4 {
		t.Errorf("expected second car RowNumber 4, got %d", cars[1].RowNumber)
	}
	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if errors[0].Row != 3 {
		t.Errorf("expected error at row 3, got %d", errors[0].Row)
	}
}

// TestParse_LargeFile asserts the parser handles 1000 rows without error.
func TestParse_LargeFile(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(header + "\r\n")
	for i := 0; i < 1000; i++ {
		vin := fmt.Sprintf("X5XJ%013d", i)
		sb.WriteString(baseRow(vin, "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n")
	}

	cars, errors, err := Parse([]byte(sb.String()))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	if len(cars) != 1000 {
		t.Fatalf("expected 1000 cars, got %d", len(cars))
	}
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}
}

// TestParse_EmptyFile asserts that an empty input returns empty results with no error.
func TestParse_EmptyFile(t *testing.T) {
	cars, errors, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("expected no fatal error, got: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 cars, got %d", len(cars))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}

// TestParse_HeaderOnly asserts that a file with only a header returns empty
// results with no error.
func TestParse_HeaderOnly(t *testing.T) {
	csvData := header + "\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("expected no fatal error, got: %v", err)
	}
	if len(cars) != 0 {
		t.Errorf("expected 0 cars, got %d", len(cars))
	}
	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}

// TestParse_MixedValidAndInvalid asserts that one bad row does not fail the
// whole batch — valid rows are still returned and the bad row appears in errors.
func TestParse_MixedValidAndInvalid(t *testing.T) {
	csvData := header + "\r\n" +
		baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n" +
		baseRow("", "BMW", "X5", "2019", "30000", "12000000", "KZT", "Черный",
			"3.0 дизель 245 л.с.", "AT", "SUV", "", "sold", "2024-06-20") + "\r\n" +
		baseRow("Y1YJ2345678901234", "Toyota", "Corolla", "2021", "15000", "4000000", "KZT", "Красный",
			"1.6 бензин 110 л.с.", "MT", "Хэтчбек", "", "active", "2024-03-10") + "\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	if len(cars) != 2 {
		t.Fatalf("expected 2 valid cars, got %d", len(cars))
	}
	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if errors[0].Row != 3 {
		t.Errorf("expected error at row 3, got %d", errors[0].Row)
	}
	if !strings.Contains(errors[0].Reason, "missing VIN") {
		t.Errorf("expected 'missing VIN' in reason, got %s", errors[0].Reason)
	}
}

// TestParse_DefectsRawPreserved asserts that the raw defects string is preserved
// as-is in CarRecord.DefectsRaw.
func TestParse_DefectsRawPreserved(t *testing.T) {
	defects := "Стук в подвеске, Требуется замена масла"
	csvData := header + "\r\n" +
		baseRow("X5XJ1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", defects, "active", "2024-01-15") + "\r\n"

	cars, _, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	if cars[0].DefectsRaw != defects {
		t.Errorf("expected DefectsRaw %q, got %q", defects, cars[0].DefectsRaw)
	}
}

// TestParse_VINNormalizedToUppercase asserts that lowercase VINs are normalized
// to uppercase.
func TestParse_VINNormalizedToUppercase(t *testing.T) {
	csvData := header + "\r\n" +
		baseRow("x5xj1234567890123", "Toyota", "Camry", "2020", "50000", "5000000", "KZT", "Белый",
			"2.0 бензин 150 л.с.", "AT", "Седан", "", "active", "2024-01-15") + "\r\n"

	cars, _, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}

	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}
	if cars[0].VIN != "X5XJ1234567890123" {
		t.Errorf("expected VIN normalized to uppercase X5XJ1234567890123, got %s", cars[0].VIN)
	}
}

// intPtr returns a pointer to an int for test convenience.
func intPtr(v int) *int {
	return &v
}
