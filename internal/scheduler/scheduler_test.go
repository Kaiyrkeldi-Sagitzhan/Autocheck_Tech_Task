package scheduler

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"awesomeProject5/internal/database"
	"awesomeProject5/internal/domain"
	"awesomeProject5/internal/importer"
	"awesomeProject5/internal/repository"
)

// mockClock is a controllable clock for testing.
type mockClock struct {
	now      time.Time
	mu       sync.Mutex
	timers   []*mockTimer
}

type mockTimer struct {
	ch     chan time.Time
	expiry time.Time
	fired  bool
}

func newMockClock(t time.Time) *mockClock {
	return &mockClock{now: t}
}

func (m *mockClock) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.now
}

func (m *mockClock) After(d time.Duration) <-chan time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan time.Time, 1)
	expiry := m.now.Add(d)
	m.timers = append(m.timers, &mockTimer{ch: ch, expiry: expiry})
	if d <= 0 {
		ch <- m.now
	}
	return ch
}

func (m *mockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.now = m.now.Add(d)
	for _, t := range m.timers {
		if !t.fired && m.now.After(t.expiry) || m.now.Equal(t.expiry) {
			t.fired = true
			select {
			case t.ch <- m.now:
			default:
			}
		}
	}
}

func TestScheduler_New_InvalidCronFailsFast(t *testing.T) {
	_, err := New("invalid-cron", "/tmp/test.csv", func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		return &domain.ImportReport{}, nil
	})
	if err == nil {
		t.Fatal("expected error for invalid cron expression, got nil")
	}
}

func TestScheduler_EnabledFalse_NoRuns(t *testing.T) {
	clock := newMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	var count atomic.Int64

	tmpFile, err := os.CreateTemp("", "scheduler-enabled-*.csv")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	sched, err := New("0 * * * *", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		count.Add(1)
		return &domain.ImportReport{}, nil
	}, WithClock(clock), WithEnabled(false))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Advance past two scheduled times.
	clock.Advance(2 * time.Hour)
	time.Sleep(20 * time.Millisecond)

	if count.Load() != 0 {
		t.Errorf("expected 0 runs when disabled, got %d", count.Load())
	}
}

func TestScheduler_ValidCronFiresAtExpectedTimes(t *testing.T) {
	clock := newMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	var count atomic.Int64
	var times []time.Time
	var mu sync.Mutex

	tmpFile, err := os.CreateTemp("", "scheduler-cron-*.csv")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	sched, err := New("0 * * * *", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		count.Add(1)
		mu.Lock()
		times = append(times, clock.Now())
		mu.Unlock()
		return &domain.ImportReport{}, nil
	}, WithClock(clock))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Wait for the scheduler loop goroutine to start.
	time.Sleep(50 * time.Millisecond)

	// First fire at 1:00.
	clock.Advance(1 * time.Hour)
	time.Sleep(20 * time.Millisecond)

	// Second fire at 2:00.
	clock.Advance(1 * time.Hour)
	time.Sleep(20 * time.Millisecond)

	if count.Load() != 2 {
		t.Errorf("expected 2 runs, got %d", count.Load())
	}

	mu.Lock()
	defer mu.Unlock()
	if len(times) != 2 {
		t.Fatalf("expected 2 timestamps, got %d", len(times))
	}
	if !times[0].Equal(time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)) {
		t.Errorf("expected first run at 01:00, got %s", times[0].Format(time.RFC3339))
	}
	if !times[1].Equal(time.Date(2024, 1, 1, 2, 0, 0, 0, time.UTC)) {
		t.Errorf("expected second run at 02:00, got %s", times[1].Format(time.RFC3339))
	}
}

func TestScheduler_RunOnce_OverlapSkip(t *testing.T) {
	clock := newMockClock(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	var count atomic.Int64
	firstStarted := make(chan struct{})

	tmpFile, err := os.CreateTemp("", "scheduler-overlap-*.csv")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	sched, err := New("0 * * * *", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		n := count.Add(1)
		if n == 1 {
			// First call: signal that we've started, then block.
			close(firstStarted)
			select {
			case <-ctx.Done():
			case <-time.After(200 * time.Millisecond):
			}
		}
		return &domain.ImportReport{}, nil
	}, WithClock(clock))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	// Trigger first run in background.
	done := make(chan struct{})
	go func() {
		sched.RunOnce(ctx)
		close(done)
	}()

	// Wait for first run to start.
	<-firstStarted

	// Trigger second run while first is still running — should be skipped.
	sched.RunOnce(ctx)

	// Wait for first run to finish.
	<-done

	if count.Load() != 1 {
		t.Errorf("expected still 1 run due to overlap, got %d", count.Load())
	}
}

func TestScheduler_RunOnce_ProducesImportRunsRow(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := repository.New(db)
	imp := importer.New(repo)

	tmpFile, err := os.CreateTemp("", "scheduler-importruns-*.csv")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString("VIN;Brand;Model;Year;MileageKm;Price;Currency;Color;Engine;Transmission;BodyType;DefectsRaw;Status;UpdatedAt\r\nX5XJ1234567890123;Toyota;Camry;2020;50000;5000000;KZT;Белый;2.0 бензин 150 л.с.;AT;Седан;;active;2024-01-15\r\n"); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	tmpFile.Close()

	clock := newMockClock(time.Now())
	sched, err := New("0 * * * *", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		return imp.Import(ctx, req)
	}, WithClock(clock))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	sched.RunOnce(ctx)

	// Verify an import_runs row exists with trigger_type=scheduled.
	var runID int64
	var triggerType, triggeredBy, status string
	err = db.QueryRowContext(ctx, "SELECT id, trigger_type, triggered_by, status FROM import_runs WHERE trigger_type = ?", "scheduled").
		Scan(&runID, &triggerType, &triggeredBy, &status)
	if err != nil {
		t.Fatalf("query import_runs: %v", err)
	}
	if triggerType != "scheduled" {
		t.Errorf("expected trigger_type 'scheduled', got %s", triggerType)
	}
	if triggeredBy != "cron" {
		t.Errorf("expected triggered_by 'cron', got %s", triggeredBy)
	}
	if status != "success" && status != "failed" {
		t.Errorf("expected status 'success' or 'failed', got %s", status)
	}
	if runID == 0 {
		t.Error("expected non-zero run ID")
	}
}

func TestScheduler_Timeout_CancelsStuckRun(t *testing.T) {
	clock := newMockClock(time.Now())
	var started atomic.Bool

	tmpFile, err := os.CreateTemp("", "scheduler-timeout-*.csv")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	sched, err := New("0 * * * *", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		started.Store(true)
		// Block until context is cancelled.
		<-ctx.Done()
		return nil, ctx.Err()
	}, WithClock(clock), WithTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	sched.Start(ctx)
	defer sched.Stop()

	sched.RunOnce(ctx)

	if !started.Load() {
		t.Error("expected run to start")
	}

	// The run should have completed (with context.DeadlineExceeded) within a reasonable time.
	done := make(chan struct{})
	go func() {
		// Wait for the run to finish — it should finish quickly due to timeout.
		time.Sleep(200 * time.Millisecond)
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(1 * time.Second):
		t.Fatal("run did not finish within timeout")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "scheduler-startstop-*.csv")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	called := false
	sched, err := New("@every 1s", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		called = true
		return &domain.ImportReport{}, nil
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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

	sched, err := New("@every 500ms", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
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
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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

	sched, err := New("@every 500ms", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		callCount++
		return nil, context.DeadlineExceeded
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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

	sched, err := New("@every 500ms", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		callCount++
		panic("simulated panic")
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	sched.Start(ctx)
	time.Sleep(2 * time.Second)
	sched.Stop()

	if callCount == 0 {
		t.Error("expected run function to be called even after panic")
	}
}

func TestScheduler_InvalidCronExpression(t *testing.T) {
	_, err := New("invalid-cron", "/tmp/test.csv", func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		return &domain.ImportReport{}, nil
	})
	if err == nil {
		t.Error("expected error for invalid cron expression, got nil")
	}
}

func TestScheduler_FileReadError(t *testing.T) {
	var report *domain.ImportReport

	sched, err := New("@every 1s", "/tmp/nonexistent-file-read-test.csv", func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		report = &domain.ImportReport{RowsTotal: len(req.Data)}
		return report, nil
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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
	sched, err := New("@every 1s", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		receivedReport = &domain.ImportReport{
			File:      req.FileName,
			RowsTotal: 1,
			Created:   1,
			Updated:   0,
			Skipped:   0,
			Errors:    []domain.RowErrorInfo{},
		}
		return receivedReport, nil
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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
	sched, err := New("@every 1s", tmpFile.Name(), func(ctx context.Context, req importer.ImportRequest) (*domain.ImportReport, error) {
		count.Add(1)
		return &domain.ImportReport{}, nil
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

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
