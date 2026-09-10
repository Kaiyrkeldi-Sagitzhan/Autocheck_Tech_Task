// Package parser converts raw 1C export file bytes into domain CarRecords.
// Pure functions only: no I/O, no database access.
package parser

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"unicode"

	"awesomeProject5/internal/domain"
)

// Parse decodes raw file bytes (semicolon-delimited CSV) and returns parsed
// records plus per-row errors. It never crashes the whole import because of
// one malformed row.
func Parse(data []byte) ([]domain.CarRecord, []domain.RowErrorInfo, error) {
	// Strip UTF-8 BOM if present (common in Excel exports).
	text := string(data)
	if len(text) >= 3 && text[0] == 0xEF && text[1] == 0xBB && text[2] == 0xBF {
		text = text[3:]
	}

	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = ';'
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // allow variable fields

	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("read csv: %w", err)
	}

	if len(records) == 0 {
		return []domain.CarRecord{}, []domain.RowErrorInfo{}, nil
	}

	header := normalizeHeader(records[0])
	colIndex := buildColumnIndex(header)

	var cars []domain.CarRecord
	var errors []domain.RowErrorInfo

	for i, row := range records[1:] {
		rowNum := i + 2 // 1-based, header is row 1
		car, err := parseRow(row, rowNum, colIndex)
		if err != nil {
			errors = append(errors, domain.RowErrorInfo{
				Row:    rowNum,
				Reason: err.Error(),
			})
			continue
		}
		cars = append(cars, car)
	}

	return cars, errors, nil
}

// normalizeHeader lowercases and trims each column name.
func normalizeHeader(header []string) []string {
	out := make([]string, len(header))
	for i, h := range header {
		out[i] = strings.ToLower(strings.TrimSpace(h))
	}
	return out
}

// buildColumnIndex maps normalized column names to their 0-based positions.
func buildColumnIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}
	return idx
}

// getField returns the trimmed value for a column, or empty string if missing.
func getField(row []string, idx map[string]int, col string) string {
	pos, ok := idx[col]
	if !ok || pos >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[pos])
}

// parseInt safely parses a string to *int. Returns nil on failure.
func parseInt(s string) *int {
	if s == "" {
		return nil
	}
	// Remove spaces used as thousands separators (e.g. "2 500 000")
	clean := strings.ReplaceAll(s, " ", "")
	clean = strings.ReplaceAll(clean, "\u00a0", "") // non-breaking space
	v, err := strconv.Atoi(clean)
	if err != nil {
		return nil
	}
	return &v
}

// validateVIN checks that VIN is exactly 17 chars, contains only alphanumeric
// characters, and has no I, O, Q.
func validateVIN(vin string) error {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	if len(vin) != 17 {
		return fmt.Errorf("invalid VIN length: %d (expected 17)", len(vin))
	}
	for _, c := range vin {
		if c == 'I' || c == 'O' || c == 'Q' {
			return fmt.Errorf("invalid VIN character: %c (I, O, Q not allowed)", c)
		}
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return fmt.Errorf("invalid VIN character: %c (only A-Z, 0-9 allowed)", c)
		}
	}
	return nil
}

// parseRow converts a single CSV row into a CarRecord or returns an error.
func parseRow(row []string, rowNum int, idx map[string]int) (domain.CarRecord, error) {
	var car domain.CarRecord
	car.RowNumber = rowNum

	// Required fields
	vin := getField(row, idx, "vin")
	if vin == "" {
		return car, fmt.Errorf("missing VIN")
	}
	if err := validateVIN(vin); err != nil {
		return car, fmt.Errorf("VIN validation: %w", err)
	}
	car.VIN = strings.ToUpper(vin)

	brand := getField(row, idx, "brand")
	if brand == "" {
		return car, fmt.Errorf("missing brand")
	}
	car.Brand = brand

	model := getField(row, idx, "model")
	if model == "" {
		return car, fmt.Errorf("missing model")
	}
	car.Model = model

	// Optional numeric fields
	car.Year = parseInt(getField(row, idx, "year"))
	car.MileageKm = parseInt(getField(row, idx, "mileagekm"))
	car.Price = parseInt(getField(row, idx, "price"))

	// Validate year range if present
	if car.Year != nil {
		y := *car.Year
		if y < 1900 || y > 2100 {
			return car, fmt.Errorf("invalid year: %d", y)
		}
	}

	// Validate mileage if present
	if car.MileageKm != nil {
		m := *car.MileageKm
		if m < 0 {
			return car, fmt.Errorf("invalid mileage: %d", m)
		}
	}

	// String fields
	car.Currency = getField(row, idx, "currency")
	if car.Currency == "" {
		car.Currency = "KZT"
	}
	car.Color = getField(row, idx, "color")
	car.Engine = getField(row, idx, "engine")
	car.Transmission = getField(row, idx, "transmission")
	car.BodyType = getField(row, idx, "bodytype")
	car.DefectsRaw = getField(row, idx, "defectsraw")

	// Status
	statusStr := strings.ToLower(getField(row, idx, "status"))
	switch statusStr {
	case "active", "":
		car.Status = domain.CarStatusActive
	case "sold":
		car.Status = domain.CarStatusSold
	case "reserved":
		car.Status = domain.CarStatusReserved
	default:
		car.Status = domain.CarStatusActive
	}

	// UpdatedAt
	car.UpdatedAt = getField(row, idx, "updatedat")

	return car, nil
}

// seededRand is used by the dataset generator for deterministic output.
var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// Seed sets the random seed for deterministic dataset generation.
func Seed(seed int64) {
	seededRand.Seed(seed)
}
