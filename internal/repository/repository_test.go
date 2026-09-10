package repository

import (
	"context"
	"testing"
	"time"

	"awesomeProject5/internal/database"
	"awesomeProject5/internal/domain"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	return New(db)
}

func TestUpsertCar_InsertNew(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	car := &domain.Car{
		VIN:          "X5XJ1234567890123",
		Brand:        "Toyota",
		Model:        "Camry",
		Year:         intPtr(2020),
		MileageKm:    intPtr(50000),
		Price:        intPtr(5000000),
		Currency:     "KZT",
		Color:        "Белый",
		Engine:       "2.0 бензин 150 л.с.",
		Transmission: "AT",
		BodyType:     "Седан",
		Defects:      []string{},
		DefectsRaw:   "",
		Status:       domain.CarStatusActive,
		SourceFile:   "test.csv",
	}

	created, err := repo.UpsertCar(ctx, car)
	if err != nil {
		t.Fatalf("UpsertCar: %v", err)
	}
	if !created {
		t.Error("expected created=true for new VIN")
	}

	// Verify car exists.
	fetched, err := repo.CarByVIN(ctx, car.VIN)
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected car to exist after insert")
	}
	if fetched.Brand != "Toyota" {
		t.Errorf("expected Brand Toyota, got %s", fetched.Brand)
	}
	if fetched.Model != "Camry" {
		t.Errorf("expected Model Camry, got %s", fetched.Model)
	}
	if fetched.Year == nil || *fetched.Year != 2020 {
		t.Errorf("expected Year 2020, got %v", fetched.Year)
	}
	if fetched.MileageKm == nil || *fetched.MileageKm != 50000 {
		t.Errorf("expected MileageKm 50000, got %v", fetched.MileageKm)
	}
	if fetched.Price == nil || *fetched.Price != 5000000 {
		t.Errorf("expected Price 5000000, got %v", fetched.Price)
	}
	if fetched.Status != domain.CarStatusActive {
		t.Errorf("expected Status active, got %s", fetched.Status)
	}
}

func TestUpsertCar_UpdateExisting(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	// Insert initial car.
	car := &domain.Car{
		VIN:          "Y1YJ2345678901234",
		Brand:        "BMW",
		Model:        "X5",
		Year:         intPtr(2019),
		MileageKm:    intPtr(30000),
		Price:        intPtr(12000000),
		Currency:     "KZT",
		Color:        "Черный",
		Engine:       "3.0 дизель 245 л.с.",
		Transmission: "AT",
		BodyType:     "SUV",
		Defects:      []string{},
		DefectsRaw:   "",
		Status:       domain.CarStatusActive,
		SourceFile:   "test.csv",
	}
	created, err := repo.UpsertCar(ctx, car)
	if err != nil {
		t.Fatalf("first UpsertCar: %v", err)
	}
	if !created {
		t.Error("expected created=true for first insert")
	}

	// Update the same car.
	car.MileageKm = intPtr(35000)
	car.Price = intPtr(11500000)
	car.DefectsRaw = "Царапина на бампере"
	car.Defects = []string{"Царапина на бампере"}

	created, err = repo.UpsertCar(ctx, car)
	if err != nil {
		t.Fatalf("second UpsertCar: %v", err)
	}
	if created {
		t.Error("expected created=false for update")
	}

	// Verify updates.
	fetched, err := repo.CarByVIN(ctx, car.VIN)
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if fetched.MileageKm == nil || *fetched.MileageKm != 35000 {
		t.Errorf("expected MileageKm 35000, got %v", fetched.MileageKm)
	}
	if fetched.Price == nil || *fetched.Price != 11500000 {
		t.Errorf("expected Price 11500000, got %v", fetched.Price)
	}
	if len(fetched.Defects) != 1 || fetched.Defects[0] != "Царапина на бампере" {
		t.Errorf("expected defects [Царапина на бампере], got %v", fetched.Defects)
	}
}

func TestUpsertCar_SameVINTwice_NoDuplicate(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	car1 := &domain.Car{
		VIN:          "Z1ZJ3456789012345",
		Brand:        "Toyota",
		Model:        "Corolla",
		Year:         intPtr(2021),
		MileageKm:    intPtr(15000),
		Price:        intPtr(4000000),
		Currency:     "KZT",
		Color:        "Красный",
		Engine:       "1.6 бензин 110 л.с.",
		Transmission: "MT",
		BodyType:     "Хэтчбек",
		Defects:      []string{},
		DefectsRaw:   "",
		Status:       domain.CarStatusActive,
		SourceFile:   "test1.csv",
	}

	_, err := repo.UpsertCar(ctx, car1)
	if err != nil {
		t.Fatalf("first UpsertCar: %v", err)
	}

	car2 := &domain.Car{
		VIN:          "Z1ZJ3456789012345",
		Brand:        "Toyota",
		Model:        "Corolla",
		Year:         intPtr(2022),
		MileageKm:    intPtr(20000),
		Price:        intPtr(4200000),
		Currency:     "KZT",
		Color:        "Синий",
		Engine:       "1.6 бензин 110 л.с.",
		Transmission: "MT",
		BodyType:     "Хэтчбек",
		Defects:      []string{},
		DefectsRaw:   "",
		Status:       domain.CarStatusSold,
		SourceFile:   "test2.csv",
	}

	_, err = repo.UpsertCar(ctx, car2)
	if err != nil {
		t.Fatalf("second UpsertCar: %v", err)
	}

	// Verify only one car exists.
	cars, err := repo.CarByVIN(ctx, car1.VIN)
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if cars == nil {
		t.Fatal("expected car to exist")
	}
	if cars.Brand != "Toyota" {
		t.Errorf("expected Brand Toyota, got %s", cars.Brand)
	}
	if cars.Status != domain.CarStatusSold {
		t.Errorf("expected Status sold, got %s", cars.Status)
	}
	if cars.MileageKm == nil || *cars.MileageKm != 20000 {
		t.Errorf("expected MileageKm 20000, got %v", cars.MileageKm)
	}
}

func TestUpsertCar_UpdateMileagePriceDefects(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	car := &domain.Car{
		VIN:          "A1AJ4567890123456",
		Brand:        "Hyundai",
		Model:        "Tucson",
		Year:         intPtr(2023),
		MileageKm:    intPtr(10000),
		Price:        intPtr(8000000),
		Currency:     "KZT",
		Color:        "Серебристый",
		Engine:       "2.0 бензин 150 л.с.",
		Transmission: "AT",
		BodyType:     "Кроссовер",
		Defects:      []string{},
		DefectsRaw:   "",
		Status:       domain.CarStatusActive,
		SourceFile:   "test.csv",
	}

	_, err := repo.UpsertCar(ctx, car)
	if err != nil {
		t.Fatalf("first UpsertCar: %v", err)
	}

	// Update mileage, price, and defects.
	car.MileageKm = intPtr(25000)
	car.Price = intPtr(7500000)
	car.DefectsRaw = "Стук в подвеске, Требуется замена масла"
	car.Defects = []string{"Стук в подвеске", "Требуется замена масла"}

	_, err = repo.UpsertCar(ctx, car)
	if err != nil {
		t.Fatalf("second UpsertCar: %v", err)
	}

	fetched, err := repo.CarByVIN(ctx, car.VIN)
	if err != nil {
		t.Fatalf("CarByVIN: %v", err)
	}
	if fetched.MileageKm == nil || *fetched.MileageKm != 25000 {
		t.Errorf("expected MileageKm 25000, got %v", fetched.MileageKm)
	}
	if fetched.Price == nil || *fetched.Price != 7500000 {
		t.Errorf("expected Price 7500000, got %v", fetched.Price)
	}
	if len(fetched.Defects) != 2 {
		t.Errorf("expected 2 defects, got %d: %v", len(fetched.Defects), fetched.Defects)
	}
}

func TestCreateAndFinishImportRun(t *testing.T) {
	repo := setupTestDB(t)
	ctx := context.Background()

	run := &domain.ImportRun{
		TriggerType: "manual",
		FileName:    "test.csv",
		Status:      "running",
		RowsTotal:   10,
		StartedAt:   time.Now().UTC(),
	}

	if err := repo.CreateImportRun(ctx, run); err != nil {
		t.Fatalf("CreateImportRun: %v", err)
	}
	if run.ID == 0 {
		t.Error("expected run.ID to be set after CreateImportRun")
	}

	// Record an error.
	if err := repo.RecordImportError(ctx, run.ID, 2, "missing VIN", ""); err != nil {
		t.Fatalf("RecordImportError: %v", err)
	}

	// Finish the run.
	run.Created = 9
	run.Updated = 0
	run.Skipped = 1
	now := time.Now().UTC()
	run.FinishedAt = &now

	if err := repo.FinishImportRun(ctx, run.ID, "failed", run); err != nil {
		t.Fatalf("FinishImportRun: %v", err)
	}

	// Verify by querying directly.
	var status string
	var rowsTotal, created, skipped int
	err := repo.db.QueryRowContext(ctx,
		"SELECT status, rows_total, created, skipped FROM import_runs WHERE id = ?", run.ID,
	).Scan(&status, &rowsTotal, &created, &skipped)
	if err != nil {
		t.Fatalf("query import_run: %v", err)
	}
	if status != "failed" {
		t.Errorf("expected status failed, got %s", status)
	}
	if rowsTotal != 10 {
		t.Errorf("expected rows_total 10, got %d", rowsTotal)
	}
	if created != 9 {
		t.Errorf("expected created 9, got %d", created)
	}
	if skipped != 1 {
		t.Errorf("expected skipped 1, got %d", skipped)
	}
}

func intPtr(v int) *int {
	return &v
}
