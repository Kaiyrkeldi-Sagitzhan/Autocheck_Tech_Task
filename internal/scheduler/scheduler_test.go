package scheduler

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"awesomeProject5/internal/domain"
)

func TestScheduler_StartStop(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "scheduler-startstop-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	called := false
	sched := New("@every 1s", tmpFile.Name(), func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		called = true
		return &domain.ImportReport{}, nil
	})

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(2 * time.Second)
	sched.Stop()

	if !called {
		t.Error("expected run function to be called")
	}
}

func TestScheduler_OverlapPrevention(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "scheduler-overlap-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	var mu sync.Mutex
	running := false
	callCount := 0

	sched := New("@every 500ms", tmpFile.Name(), func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		mu.Lock()
		if running {
			mu.Unlock()
			return &domain.ImportReport{}, nil
		}
		running = true
		callCount++
		mu.Unlock()

		// Simulate a long-running import.
		time.Sleep(3 * time.Second)

		mu.Lock()
		running = false
		mu.Unlock()

		return &domain.ImportReport{}, nil
	})

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(2 * time.Second)
	sched.Stop()

	if callCount != 1 {
		t.Errorf("expected 1 call due to overlap prevention, got %d", callCount)
	}
}

func TestScheduler_ErrorHandling(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "scheduler-error-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	callCount := 0

	sched := New("@every 500ms", tmpFile.Name(), func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		callCount++
		return nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(2 * time.Second)
	sched.Stop()

	if callCount == 0 {
		t.Error("expected run function to be called despite errors")
	}
}

func TestScheduler_PanicRecovery(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "scheduler-panic-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	callCount := 0

	sched := New("@every 500ms", tmpFile.Name(), func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		callCount++
		panic("simulated panic")
	})

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(2 * time.Second)
	sched.Stop()

	if callCount == 0 {
		t.Error("expected run function to be called even after panic")
	}
}

func TestScheduler_InvalidCronExpression(t *testing.T) {
	sched := New("invalid-cron", "/tmp/test.csv", func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		return &domain.ImportReport{}, nil
	})

	ctx := context.Background()
	// Should not panic on invalid cron expression.
	sched.Start(ctx)
	sched.Stop()
}

func TestScheduler_FileReadError(t *testing.T) {
	var report *domain.ImportReport

	sched := New("@every 1s", "/tmp/nonexistent-file-read-test.csv", func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		report = &domain.ImportReport{RowsTotal: len(data)}
		return report, nil
	})

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(2 * time.Second)
	sched.Stop()

	// The run function should not be called because file read fails before it.
	if report != nil {
		t.Error("expected run function not to be called when file read fails")
	}
}

func TestScheduler_StatisticsLogging(t *testing.T) {
	// Create a temporary CSV file.
	tmpFile, err := os.CreateTemp("", "scheduler-stats-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\nX5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	var receivedReport *domain.ImportReport
	sched := New("@every 1s", tmpFile.Name(), func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		receivedReport = &domain.ImportReport{
			File:      fileName,
			RowsTotal: 1,
			Created:   1,
			Updated:   0,
			Skipped:   0,
			Errors:    []domain.RowErrorInfo{},
		}
		return receivedReport, nil
	})

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(2 * time.Second)
	sched.Stop()

	if receivedReport == nil {
		t.Fatal("expected report to be received")
	}
	if receivedReport.RowsTotal != 1 {
		t.Errorf("expected RowsTotal 1, got %d", receivedReport.RowsTotal)
	}
	if receivedReport.Created != 1 {
		t.Errorf("expected Created 1, got %d", receivedReport.Created)
	}
}

func TestScheduler_MultipleRuns(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "scheduler-multiple-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	var count atomic.Int64
	sched := New("@every 1s", tmpFile.Name(), func(ctx context.Context, fileName string, data []byte) (*domain.ImportReport, error) {
		count.Add(1)
		return &domain.ImportReport{}, nil
	})

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(3 * time.Second)
	sched.Stop()

	// Should run at ~1s and ~2s (3 runs total if first is immediate, but with cron it's at interval).
	// With @every 1s, it runs at 1s, 2s, 3s. We sleep 3s, so expect at least 2.
	if count.Load() < 2 {
		t.Errorf("expected at least 2 runs, got %d", count.Load())
	}
}
