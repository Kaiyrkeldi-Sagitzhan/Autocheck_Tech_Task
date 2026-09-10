package parser

import (
	"strings"
	"testing"

	"awesomeProject5/internal/domain"
)

func TestParseValidCSV(t *testing.T) {
	csvData := strings.Join([]string{
		"VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt",
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15",
		"Y1YJ2345678901234;BMW;X5;2019;30000;12000000;KZT;Черный;3.0 дизель 245 л.с.;AT;SUV;Царапина на бампере;sold;2024-06-20",
		"", // empty line should be skipped
	}, "\r\n")

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 2 {
		t.Fatalf("expected 2 cars, got %d", len(cars))
	}

	// First car
	c1 := cars[0]
	if c1.VIN != "X5XJ1234567890123" {
		t.Errorf("expected VIN X5XJ1234567890123, got %s", c1.VIN)
	}
	if c1.Brand != "Toyota" {
		t.Errorf("expected Brand Toyota, got %s", c1.Brand)
	}
	if c1.Model != "Camry" {
		t.Errorf("expected Model Camry, got %s", c1.Model)
	}
	if c1.Year == nil || *c1.Year != 2020 {
		t.Errorf("expected Year 2020, got %v", c1.Year)
	}
	if c1.MileageKm == nil || *c1.MileageKm != 50000 {
		t.Errorf("expected MileageKm 50000, got %v", c1.MileageKm)
	}
	if c1.Price == nil || *c1.Price != 5000000 {
		t.Errorf("expected Price 5000000, got %v", c1.Price)
	}
	if c1.Currency != "KZT" {
		t.Errorf("expected Currency KZT, got %s", c1.Currency)
	}
	if c1.Color != "Белый" {
		t.Errorf("expected Color Белый, got %s", c1.Color)
	}
	if c1.Status != domain.CarStatusActive {
		t.Errorf("expected Status active, got %s", c1.Status)
	}
	if c1.UpdatedAt != "2024-01-15" {
		t.Errorf("expected UpdatedAt 2024-01-15, got %s", c1.UpdatedAt)
	}
	if c1.RowNumber != 2 {
		t.Errorf("expected RowNumber 2, got %d", c1.RowNumber)
	}

	// Second car
	c2 := cars[1]
	if c2.Status != domain.CarStatusSold {
		t.Errorf("expected Status sold, got %s", c2.Status)
	}
	if c2.DefectsRaw != "Царапина на бампере" {
		t.Errorf("expected DefectsRaw 'Царапина на бампере', got %s", c2.DefectsRaw)
	}

	if len(errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errors))
	}
}

func TestParseMissingVIN(t *testing.T) {
	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 0 {
		t.Fatalf("expected 0 cars, got %d", len(cars))
	}

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	if errors[0].Row != 2 {
		t.Errorf("expected error at row 2, got %d", errors[0].Row)
	}
	if !strings.Contains(errors[0].Reason, "missing VIN") {
		t.Errorf("expected 'missing VIN' in reason, got %s", errors[0].Reason)
	}
}

func TestParseInvalidVIN(t *testing.T) {
	cases := []struct {
		name string
		vin  string
		want string
	}{
		{"too short", "ABC123", "invalid VIN length"},
		{"too long", "ABCDEFGHIJKLMNOPQRS0", "invalid VIN length"},
		{"contains I", "ABCDEFGHIJKLMNOPQ", "invalid VIN character"},
		{"contains O", "ABCDEFGHIJKLMNOP0", "invalid VIN character"},
		{"contains Q", "ABCDEFGHIJKLMNQPQ", "invalid VIN character"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
				tc.vin + ";Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

			cars, errors, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(cars) != 0 {
				t.Fatalf("expected 0 cars, got %d", len(cars))
			}

			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d", len(errors))
			}

			if !strings.Contains(errors[0].Reason, tc.want) {
				t.Errorf("expected reason to contain %q, got %s", tc.want, errors[0].Reason)
			}
		})
	}
}

func TestParseInvalidMileage(t *testing.T) {
	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;abc;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	if cars[0].MileageKm != nil {
		t.Errorf("expected nil MileageKm, got %d", *cars[0].MileageKm)
	}

	if len(errors) != 0 {
		t.Fatalf("expected 0 errors for non-fatal invalid mileage, got %d", len(errors))
	}
}

func TestParseInvalidYear(t *testing.T) {
	cases := []struct {
		name  string
		year  string
		fatal bool
	}{
		{"year 1800", "1800", true},
		{"year 2101", "2101", true},
		{"year abc", "abc", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
				"X5XJ1234567890123;Toyota;Camry;" + tc.year + ";50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

			cars, errors, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.fatal {
				if len(cars) != 0 {
					t.Fatalf("expected 0 cars for fatal error, got %d", len(cars))
				}
				if len(errors) != 1 {
					t.Fatalf("expected 1 error, got %d", len(errors))
				}
				if !strings.Contains(errors[0].Reason, "invalid year") {
					t.Errorf("expected 'invalid year' in reason, got %s", errors[0].Reason)
				}
			} else {
				if len(cars) != 1 {
					t.Fatalf("expected 1 car for non-fatal error, got %d", len(cars))
				}
				if cars[0].Year != nil {
					t.Errorf("expected nil Year, got %d", *cars[0].Year)
				}
			}
		})
	}
}

func TestParseMissingRequiredFields(t *testing.T) {
	cases := []struct {
		name     string
		row      string
		expected string
	}{
		{"missing brand", "X5XJ1234567890123;;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15", "missing brand"},
		{"missing model", "X5XJ1234567890123;Toyota;;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15", "missing model"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" + tc.row + "\r\n"

			cars, errors, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(cars) != 0 {
				t.Fatalf("expected 0 cars, got %d", len(cars))
			}

			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d", len(errors))
			}

			if !strings.Contains(errors[0].Reason, tc.expected) {
				t.Errorf("expected reason to contain %q, got %s", tc.expected, errors[0].Reason)
			}
		})
	}
}

func TestParseMalformedColumns(t *testing.T) {
	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"SHORT;Row\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 0 {
		t.Fatalf("expected 0 cars, got %d", len(cars))
	}

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	if !strings.Contains(errors[0].Reason, "invalid VIN length") {
		t.Errorf("expected 'invalid VIN length' in reason, got %s", errors[0].Reason)
	}
}

func TestParseMixedValidAndInvalid(t *testing.T) {
	csvData := strings.Join([]string{
		"VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt",
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15",
		";BMW;X5;2019;30000;12000000;KZT;Черный;3.0 дизель 245 л.с.;AT;SUV;;sold;2024-06-20",
		"Y1YJ2345678901234;Toyota;Corolla;2021;15000;4000000;KZT;Красный;1.6 бензин 110 л.с.;MT;Хэтчбек;;active;2024-03-10",
	}, "\r\n")

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 2 {
		t.Fatalf("expected 2 cars, got %d", len(cars))
	}

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	if errors[0].Row != 3 {
		t.Errorf("expected error at row 3, got %d", errors[0].Row)
	}
}

func TestParseEmptyFile(t *testing.T) {
	cars, errors, err := Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 0 {
		t.Fatalf("expected 0 cars, got %d", len(cars))
	}
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}
}

func TestParseHeaderOnly(t *testing.T) {
	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 0 {
		t.Fatalf("expected 0 cars, got %d", len(cars))
	}
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}
}

func TestParseWhitespaceTrimming(t *testing.T) {
	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"  X5XJ1234567890123  ;  Toyota  ;  Camry  ;  2020  ;  50000  ;  5000000  ;  KZT  ;  Белый  ;  2.0 бензин 150 л.с.  ;  AT  ;  Седан  ;  ;  active  ;  2024-01-15  \r\n"

	cars, _, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	if cars[0].VIN != "X5XJ1234567890123" {
		t.Errorf("expected trimmed VIN, got %q", cars[0].VIN)
	}
	if cars[0].Brand != "Toyota" {
		t.Errorf("expected trimmed Brand, got %q", cars[0].Brand)
	}
}

func TestParseDefaultCurrency(t *testing.T) {
	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	cars, _, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 1 {
		t.Fatalf("expected 1 car, got %d", len(cars))
	}

	if cars[0].Currency != "KZT" {
		t.Errorf("expected default Currency KZT, got %s", cars[0].Currency)
	}
}

func TestParseStatusMapping(t *testing.T) {
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
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
				"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;" + tc.input + ";2024-01-15\r\n"

			cars, _, err := Parse([]byte(csvData))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
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

func TestParseNegativeMileage(t *testing.T) {
	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;-100;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 0 {
		t.Fatalf("expected 0 cars for negative mileage, got %d", len(cars))
	}

	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}

	if !strings.Contains(errors[0].Reason, "invalid mileage") {
		t.Errorf("expected 'invalid mileage' in reason, got %s", errors[0].Reason)
	}
}

func TestParseRowNumberTracking(t *testing.T) {
	csvData := strings.Join([]string{
		"VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt",
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15",
		";BMW;X5;2019;30000;12000000;KZT;Черный;3.0 дизель 245 л.с.;AT;SUV;;sold;2024-06-20",
		"Y1YJ2345678901234;Toyota;Corolla;2021;15000;4000000;KZT;Красный;1.6 бензин 110 л.с.;MT;Хэтчбек;;active;2024-03-10",
	}, "\r\n")

	cars, errors, err := Parse([]byte(csvData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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

	if errors[0].Row != 3 {
		t.Errorf("expected error at row 3, got %d", errors[0].Row)
	}
}

func TestParseLargeFile(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n")

	for i := 0; i < 1000; i++ {
		sb.WriteString("X5XJ123456789012")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString(";Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n")
	}

	cars, errors, err := Parse([]byte(sb.String()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cars) != 1000 {
		t.Fatalf("expected 1000 cars, got %d", len(cars))
	}
	if len(errors) != 0 {
		t.Fatalf("expected 0 errors, got %d", len(errors))
	}
}
