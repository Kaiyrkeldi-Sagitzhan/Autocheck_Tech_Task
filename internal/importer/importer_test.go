package importer

import (
	"awesomeProject5/internal/database"
	"awesomeProject5/internal/domain"
	"awesomeProject5/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
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

	report, err := imp.Import(ctx, ImportRequest{
		TriggerType: "manual",
		TriggeredBy: "api",
		FileName:    "test.csv",
		Data:        []byte(csvData),
	})
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

	report1, err := imp.Import(ctx, ImportRequest{
		TriggerType: "manual",
		TriggeredBy: "api",
		FileName:    "test1.csv",
		Data:        []byte(csv1),
	})
	if err != nil {
		t.Fatalf("first Import: %v", err)
	}
	if report1.Created != 1 {
		t.Errorf("expected Created 1, got %d", report1.Created)
	}

	// Second import with same VIN but updated fields.
	csv2 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2021;60000;5500000;KZT;Красный;2.0 бензин 150 л.с.;AT;Седан;Царапина на бампере;sold;2024-12-01\r\n"

	report2, err := imp.Import(ctx, ImportRequest{
		TriggerType: "manual",
		TriggeredBy: "api",
		FileName:    "test2.csv",
		Data:        []byte(csv2),
	})
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

	report, err := imp.Import(ctx, ImportRequest{
		TriggerType: "manual",
		TriggeredBy: "api",
		FileName:    "test.csv",
		Data:        []byte(csv),
	})
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

	report, err := imp.Import(ctx, ImportRequest{
		TriggerType: "manual",
		TriggeredBy: "api",
		FileName:    "test.csv",
		Data:        []byte(csvData),
	})
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

	report, err := imp.Import(ctx, ImportRequest{
		TriggerType: "manual",
		TriggeredBy: "api",
		FileName:    "empty.csv",
		Data:        []byte(""),
	})
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

	_, err := imp.Import(ctx, ImportRequest{
		TriggerType: "manual",
		TriggeredBy: "api",
		FileName:    "test.csv",
		Data:        []byte(csvData),
	})
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

// generateLargeCSV creates a CSV with the given number of data rows using valid 17-char VINs.
func generateLargeCSV(rows int) string {
	var sb strings.Builder
	sb.WriteString("VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n")
	for i := 0; i < rows; i++ {
		// Use a valid 17-char VIN pattern (4-char prefix + 13-digit number).
		vin := fmt.Sprintf("X5XJ%013d", i)
		sb.WriteString(fmt.Sprintf("%s;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n", vin))
	}
	return sb.String()
}

func TestImport_OverlapPrevention(t *testing.T) {
	imp, _ := setupImporter(t)
	ctx := context.Background()

	// Use a larger dataset so the import takes long enough to overlap.
	csvData := generateLargeCSV(500)

	var wg sync.WaitGroup
	var firstErr, secondErr error
	var firstReport, secondReport *domain.ImportReport

	// Use a channel to ensure the second import starts while the first is still running.
	firstStarted := make(chan struct{})

	// Start first import in background.
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(firstStarted)
		firstReport, firstErr = imp.Import(ctx, ImportRequest{
			TriggerType: "manual",
			TriggeredBy: "api",
			FileName:    "first.csv",
			Data:        []byte(csvData),
		})
	}()

	// Wait for first import to start, then immediately start second.
	<-firstStarted

	// Start second import while first is still running.
	wg.Add(1)
	go func() {
		defer wg.Done()
		secondReport, secondErr = imp.Import(ctx, ImportRequest{
			TriggerType: "manual",
			TriggeredBy: "api",
			FileName:    "second.csv",
			Data:        []byte(csvData),
		})
	}()

	wg.Wait()

	if firstErr != nil {
		t.Fatalf("first import failed: %v", firstErr)
	}
	if firstReport == nil || firstReport.Created != 500 {
		t.Errorf("expected first report with Created=500, got %+v", firstReport)
	}

	if secondErr == nil {
		t.Fatal("expected second import to fail with overlap error")
	}
	var inProgress *ImportInProgressError
	if !errors.As(secondErr, &inProgress) {
		t.Fatalf("expected ImportInProgressError, got %T: %v", secondErr, secondErr)
	}
	if secondReport != nil {
		t.Error("expected nil report for overlapping import")
	}
}

func TestImport_OverlapPrevention_ScheduledAndManual(t *testing.T) {
	imp, _ := setupImporter(t)
	ctx := context.Background()

	// Use a larger dataset so the import takes long enough to overlap.
	csvData := generateLargeCSV(500)

	var wg sync.WaitGroup
	var scheduledErr, manualErr error

	// Use a channel to ensure the second import starts while the first is still running.
	firstStarted := make(chan struct{})

	// Start scheduled import in background.
	wg.Add(1)
	go func() {
		defer wg.Done()
		close(firstStarted)
		_, scheduledErr = imp.Import(ctx, ImportRequest{
			TriggerType: "scheduled",
			TriggeredBy: "cron",
			FileName:    "scheduled.csv",
			Data:        []byte(csvData),
		})
	}()

	// Wait for first import to start, then immediately start second.
	<-firstStarted

	// Start manual import while scheduled is running.
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, manualErr = imp.Import(ctx, ImportRequest{
			TriggerType: "manual",
			TriggeredBy: "api",
			FileName:    "manual.csv",
			Data:        []byte(csvData),
		})
	}()

	wg.Wait()

	// Exactly one should succeed, the other should get ImportInProgressError.
	successCount := 0
	if scheduledErr == nil {
		successCount++
	}
	if manualErr == nil {
		successCount++
	}
	if successCount != 1 {
		t.Fatalf("expected exactly 1 success, got %d (scheduledErr=%v, manualErr=%v)", successCount, scheduledErr, manualErr)
	}

	// The one that failed should have ImportInProgressError.
	if scheduledErr != nil {
		var inProgress *ImportInProgressError
		if !errors.As(scheduledErr, &inProgress) {
			t.Fatalf("expected ImportInProgressError for scheduled, got %T: %v", scheduledErr, scheduledErr)
		}
	}
	if manualErr != nil {
		var inProgress *ImportInProgressError
		if !errors.As(manualErr, &inProgress) {
			t.Fatalf("expected ImportInProgressError for manual, got %T: %v", manualErr, manualErr)
		}
	}
}

// TestImport_MixedBatch asserts that a batch containing new, updated, and
// unchanged VINs produces correct Created/Updated/Skipped counts simultaneously.
func TestImport_MixedBatch(t *testing.T) {
	imp, repo := setupImporter(t)
	ctx := context.Background()

	// Pre-seed two cars.
	seed1 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"A1AJ4567890123456;Hyundai;Tucson;2023;10000;8000000;KZT;Серебристый;2.0 бензин 150 л.с.;AT;Кроссовер;;active;2024-01-01\r\n"
	seed2 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"B2BK5678901234567;Kia;Sportage;2022;20000;7000000;KZT;Красный;2.4 бензин 180 л.с.;AT;Кроссовер;;active;2024-02-01\r\n"

	_, err := imp.Import(ctx, ImportRequest{TriggerType: "manual", TriggeredBy: "api", FileName: "seed.csv", Data: []byte(seed1)})
	if err != nil {
		t.Fatalf("seed import 1: %v", err)
	}
	_, err = imp.Import(ctx, ImportRequest{TriggerType: "manual", TriggeredBy: "api", FileName: "seed.csv", Data: []byte(seed2)})
	if err != nil {
		t.Fatalf("seed import 2: %v", err)
	}

	// Mixed batch:
	//   - C1CK6789012345678: brand new VIN → Created
	//   - A1AJ4567890123456: existing VIN, changed mileage → Updated
	//   - B2BK5678901234567: existing VIN, identical data → Skipped
	csv := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"C1CK6789012345678;Toyota;RAV4;2024;5000;9000000;KZT;Белый;2.0 гибрид 180 л.с.;AT;Кроссовер;;active;2024-03-01\r\n" +
		"A1AJ4567890123456;Hyundai;Tucson;2023;15000;8500000;KZT;Черный;2.0 бензин 150 л.с.;AT;Кроссовер;;active;2024-01-01\r\n" +
		"B2BK5678901234567;Kia;Sportage;2022;20000;7000000;KZT;Красный;2.4 бензин 180 л.с.;AT;Кроссовер;;active;2024-02-01\r\n"

	report, err := imp.Import(ctx, ImportRequest{TriggerType: "manual", TriggeredBy: "api", FileName: "mixed.csv", Data: []byte(csv)})
	if err != nil {
		t.Fatalf("mixed import: %v", err)
	}

	if report.RowsTotal != 3 {
		t.Errorf("expected RowsTotal 3, got %d", report.RowsTotal)
	}
	if report.Created != 1 {
		t.Errorf("expected Created 1, got %d", report.Created)
	}
	if report.Updated != 1 {
		t.Errorf("expected Updated 1, got %d", report.Updated)
	}
	if report.Skipped != 1 {
		t.Errorf("expected Skipped 1, got %d", report.Skipped)
	}

	// Verify final row count = 3 distinct VINs.
	total, err := repo.CountCars(ctx)
	if err != nil {
		t.Fatalf("CountCars: %v", err)
	}
	if total != 3 {
		t.Errorf("expected 3 cars in DB, got %d", total)
	}

	// Verify the updated car has new mileage.
	car, err := repo.CarByVIN(ctx, "A1AJ4567890123456")
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if car == nil {
		t.Fatal("expected updated car to exist")
	}
	if car.MileageKm == nil || *car.MileageKm != 15000 {
		t.Errorf("expected updated MileageKm 15000, got %v", car.MileageKm)
	}
}

// TestUpsert_ReImportSameVIN_UpdatesInPlace is the critical idempotency test.
// It proves that re-importing the same VIN never creates a duplicate and
// actually updates the row in place when data changes.
func TestUpsert_ReImportSameVIN_UpdatesInPlace(t *testing.T) {
	imp, repo := setupImporter(t)
	ctx := context.Background()

	// Step 1: Import VIN X with mileage=50000.
	csv1 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	report1, err := imp.Import(ctx, ImportRequest{TriggerType: "manual", TriggeredBy: "api", FileName: "first.csv", Data: []byte(csv1)})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if report1.Created != 1 {
		t.Errorf("expected Created 1 on first import, got %d", report1.Created)
	}
	if report1.Updated != 0 {
		t.Errorf("expected Updated 0 on first import, got %d", report1.Updated)
	}

	// Step 2: Assert DB contains exactly one row with mileage=50000.
	car1, err := repo.CarByVIN(ctx, "X5XJ1234567890123")
	if err != nil {
		t.Fatalf("CarByVIN after first import: %v", err)
	}
	if car1 == nil {
		t.Fatal("expected car to exist after first import")
	}
	if car1.MileageKm == nil || *car1.MileageKm != 50000 {
		t.Errorf("expected MileageKm 50000 after first import, got %v", car1.MileageKm)
	}
	firstUpdatedAt := car1.UpdatedAt

	// Step 3: Import a SECOND batch with the SAME VIN but mileage=75000.
	csv2 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"X5XJ1234567890123;Toyota;Camry;2020;75000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"

	report2, err := imp.Import(ctx, ImportRequest{TriggerType: "manual", TriggeredBy: "api", FileName: "second.csv", Data: []byte(csv2)})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if report2.Created != 0 {
		t.Errorf("expected Created 0 on second import, got %d", report2.Created)
	}
	if report2.Updated != 1 {
		t.Errorf("expected Updated 1 on second import, got %d", report2.Updated)
	}

	// Step 4a: SELECT COUNT(*) FROM cars WHERE vin = 'X5XJ1234567890123' returns exactly 1.
	carCount, err := repo.CountCars(ctx)
	if err != nil {
		t.Fatalf("CountCars: %v", err)
	}
	if carCount != 1 {
		t.Fatalf("expected exactly 1 row for VIN X5XJ1234567890123, got %d", carCount)
	}

	// Step 4b: The single row's mileage equals 75000 (the NEW value).
	car2, err := repo.CarByVIN(ctx, "X5XJ1234567890123")
	if err != nil {
		t.Fatalf("CarByVIN after second import: %v", err)
	}
	if car2 == nil {
		t.Fatal("expected car to still exist after second import")
	}
	if car2.MileageKm == nil || *car2.MileageKm != 75000 {
		t.Errorf("expected MileageKm 75000 after second import, got %v", car2.MileageKm)
	}

	// Step 4c: Other unchanged fields preserved correctly.
	if car2.VIN != "X5XJ1234567890123" {
		t.Errorf("expected VIN X5XJ1234567890123, got %s", car2.VIN)
	}
	if car2.Brand != "Toyota" {
		t.Errorf("expected Brand Toyota, got %s", car2.Brand)
	}
	if car2.Model != "Camry" {
		t.Errorf("expected Model Camry, got %s", car2.Model)
	}
	if car2.Status != domain.CarStatusActive {
		t.Errorf("expected Status active, got %s", car2.Status)
	}

	// Step 4d: updated_at advanced between the two imports.
	if !car2.UpdatedAt.After(firstUpdatedAt) {
		t.Errorf("expected updated_at to advance: first=%s, second=%s", firstUpdatedAt, car2.UpdatedAt)
	}
}

// TestUpsert_ReImportSameVIN_DifferentTriggers is a variant of the critical
// upsert test that runs the two imports through different trigger paths
// ("manual" then "scheduled") to prove the upsert guarantee holds regardless
// of which trigger caused the import.
func TestUpsert_ReImportSameVIN_DifferentTriggers(t *testing.T) {
	imp, repo := setupImporter(t)
	ctx := context.Background()

	// First import via "manual" trigger.
	csv1 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"Y1YJ2345678901234;BMW;X5;2019;30000;12000000;KZT;Черный;3.0 дизель 245 л.с.;AT;SUV;;sold;2024-06-20\r\n"

	report1, err := imp.Import(ctx, ImportRequest{TriggerType: "manual", TriggeredBy: "api", FileName: "manual.csv", Data: []byte(csv1)})
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if report1.Created != 1 {
		t.Errorf("expected Created 1, got %d", report1.Created)
	}

	car1, err := repo.CarByVIN(ctx, "Y1YJ2345678901234")
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if car1 == nil {
		t.Fatal("expected car to exist")
	}
	firstUpdatedAt := car1.UpdatedAt

	// Second import via "scheduled" trigger with changed mileage.
	csv2 := "VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\n" +
		"Y1YJ2345678901234;BMW;X5;2019;35000;12000000;KZT;Черный;3.0 дизель 245 л.с.;AT;SUV;;sold;2024-06-20\r\n"

	report2, err := imp.Import(ctx, ImportRequest{TriggerType: "scheduled", TriggeredBy: "cron", FileName: "scheduled.csv", Data: []byte(csv2)})
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if report2.Created != 0 {
		t.Errorf("expected Created 0, got %d", report2.Created)
	}
	if report2.Updated != 1 {
		t.Errorf("expected Updated 1, got %d", report2.Updated)
	}

	// Verify no duplicate.
	carCount, err := repo.CountCars(ctx)
	if err != nil {
		t.Fatalf("CountCars: %v", err)
	}
	if carCount != 1 {
		t.Fatalf("expected exactly 1 row, got %d", carCount)
	}

	car2, err := repo.CarByVIN(ctx, "Y1YJ2345678901234")
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if car2.MileageKm == nil || *car2.MileageKm != 35000 {
		t.Errorf("expected MileageKm 35000, got %v", car2.MileageKm)
	}
	if !car2.UpdatedAt.After(firstUpdatedAt) {
		t.Errorf("expected updated_at to advance after scheduled re-import")
	}
}
