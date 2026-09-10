package importer

import (
	"context"
	"strings"
	"testing"

	"awesomeProject5/internal/database"
	"awesomeProject5/internal/domain"
	"awesomeProject5/internal/repository"
)

func setupImporter(t *testing.T) (*Importer, *repository.Repository) {
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
	imp := New(repo)
	return imp, repo
}

func TestImport_NewCars(t *testing.T) {
	imp, _ := setupImporter(t)
	ctx := context.Background()

	csvData := strings.Join([]string{
		"VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt",
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15",
		"Y1YJ2345678901234;BMW;X5;2019;30000;12000000;KZT;Черный;3.0 дизель 245 л.с.;AT;SUV;;sold;2024-06-20",
	}, "\r\n")

	report, err := imp.Import(ctx, "test.csv", []byte(csvData))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if report.RowsTotal != 2 {
		t.Errorf("expected RowsTotal 2, got %d", report.RowsTotal)
	}
	if report.Created != 2 {
		t.Errorf("expected Created 2, got %d", report.Created)
	}
	if report.Updated != 0 {
		t.Errorf("expected Updated 0, got %d", report.Updated)
	}
	if report.Skipped != 0 {
		t.Errorf("expected Skipped 0, got %d", report.Skipped)
	}
	if len(report.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(report.Errors))
	}
}

func TestImport_UpdateExistingVIN(t *testing.T) {
	imp, repo := setupImporter(t)
	ctx := context.Background()

	// First import.
	csv1 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	report1, err := imp.Import(ctx, "test1.csv", []byte(csv1))
	if err != nil {
		t.Fatalf("first Import: %v", err)
	}
	if report1.Created != 1 {
		t.Errorf("expected Created 1, got %d", report1.Created)
	}

	// Second import with same VIN but updated fields.
	csv2 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2021;60000;5500000;KZT;Красный;2.0 бензин 150 л.с.;AT;Седан;Царапина на бампере;sold;2024-12-01\r\n"

	report2, err := imp.Import(ctx, "test2.csv", []byte(csv2))
	if err != nil {
		t.Fatalf("second Import: %v", err)
	}
	if report2.Created != 0 {
		t.Errorf("expected Created 0, got %d", report2.Created)
	}
	if report2.Updated != 1 {
		t.Errorf("expected Updated 1, got %d", report2.Updated)
	}

	// Verify the car was updated.
	car, err := repo.CarByVIN(ctx, "X5XJ1234567890123")
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if car == nil {
		t.Fatal("expected car to exist")
	}
	if car.MileageKm == nil || *car.MileageKm != 60000 {
		t.Errorf("expected MileageKm 60000, got %v", car.MileageKm)
	}
	if car.Price == nil || *car.Price != 5500000 {
		t.Errorf("expected Price 5500000, got %v", car.Price)
	}
	if car.Status != domain.CarStatusSold {
		t.Errorf("expected Status sold, got %s", car.Status)
	}
	if len(car.Defects) != 1 || car.Defects[0] != "Царапина на бампере" {
		t.Errorf("expected defects [Царапина на бампере], got %v", car.Defects)
	}
}

func TestImport_SameVINTwice_NoDuplicate(t *testing.T) {
	imp, repo := setupImporter(t)
	ctx := context.Background()

	csv := strings.Join([]string{
		"VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt",
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15",
		"X5XJ1234567890123;Toyota;Camry;2021;60000;5500000;KZT;Красный;2.0 бензин 150 л.с.;AT;Седан;;sold;2024-12-01",
	}, "\r\n")

	report, err := imp.Import(ctx, "test.csv", []byte(csv))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	// First row: created, second row: updated.
	if report.Created != 1 {
		t.Errorf("expected Created 1, got %d", report.Created)
	}
	if report.Updated != 1 {
		t.Errorf("expected Updated 1, got %d", report.Updated)
	}

	// Verify only one car exists.
	car, err := repo.CarByVIN(ctx, "X5XJ1234567890123")
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if car == nil {
		t.Fatal("expected car to exist")
	}
	if car.MileageKm == nil || *car.MileageKm != 60000 {
		t.Errorf("expected MileageKm 60000, got %v", car.MileageKm)
	}
}

func TestImport_WithInvalidRows(t *testing.T) {
	imp, _ := setupImporter(t)
	ctx := context.Background()

	csvData := strings.Join([]string{
		"VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt",
		";BMW;X5;2019;30000;12000000;KZT;Черный;3.0 дизель 245 л.с.;AT;SUV;;sold;2024-06-20",
		"Y1YJ2345678901234;Toyota;Corolla;2021;15000;4000000;KZT;Красный;1.6 бензин 110 л.с.;MT;Хэтчбек;;active;2024-03-10",
	}, "\r\n")

	report, err := imp.Import(ctx, "test.csv", []byte(csvData))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if report.RowsTotal != 2 {
		t.Errorf("expected RowsTotal 2, got %d", report.RowsTotal)
	}
	if report.Created != 1 {
		t.Errorf("expected Created 1, got %d", report.Created)
	}
	if report.Skipped != 0 {
		t.Errorf("expected Skipped 0, got %d", report.Skipped)
	}
	if len(report.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(report.Errors))
	}
	if report.Errors[0].Row != 2 {
		t.Errorf("expected error at row 2, got %d", report.Errors[0].Row)
	}
	if report.Errors[0].Reason != "missing VIN" {
		t.Errorf("expected 'missing VIN' in reason, got %s", report.Errors[0].Reason)
	}
}

func TestImport_EmptyFile(t *testing.T) {
	imp, _ := setupImporter(t)
	ctx := context.Background()

	report, err := imp.Import(ctx, "empty.csv", []byte(""))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if report.RowsTotal != 0 {
		t.Errorf("expected RowsTotal 0, got %d", report.RowsTotal)
	}
	if report.Created != 0 {
		t.Errorf("expected Created 0, got %d", report.Created)
	}
}

func TestImport_DefectsNormalization(t *testing.T) {
	imp, repo := setupImporter(t)
	ctx := context.Background()

	csvData := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;Стук в подвеске, Требуется замена масла;active;2024-01-15\r\n"

	_, err := imp.Import(ctx, "test.csv", []byte(csvData))
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	car, err := repo.CarByVIN(ctx, "X5XJ1234567890123")
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if car == nil {
		t.Fatal("expected car to exist")
	}
	if len(car.Defects) != 2 {
		t.Errorf("expected 2 defects, got %d: %v", len(car.Defects), car.Defects)
	}
	if car.Defects[0] != "Стук в подвеске" {
		t.Errorf("expected first defect 'Стук в подвеске', got %s", car.Defects[0])
	}
	if car.Defects[1] != "Требуется замена масла" {
		t.Errorf("expected second defect 'Требуется замена масла', got %s", car.Defects[1])
	}
}
