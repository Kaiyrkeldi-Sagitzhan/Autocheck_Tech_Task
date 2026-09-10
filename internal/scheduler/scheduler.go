// Package scheduler runs the import on a cron schedule via a goroutine.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"awesomeProject5/internal/importer"
	"awesomeProject5/internal/domain"
	"awesomeProject5/internal/repository"
	"github.com/robfig/cron/v3"
)

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Scheduler periodically triggers imports using a cron expression.
type Scheduler struct {
	cronExpr string
	filePath string
	runFn    func(context.Context, importer.ImportRequest) (*domain.ImportReport, error)
	clock    Clock
	enabled  bool
	timeout  time.Duration
	schedule cron.Schedule
	repo     *repository.Repository // optional, for recording failed runs
	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
	inRun    atomic.Bool
}

// Option configures a Scheduler.
type Option func(*Scheduler)

// WithClock injects a custom clock (useful for tests).
func WithClock(c Clock) Option {
	return func(s *Scheduler) { s.clock = c }
}

// WithEnabled controls whether the scheduler is active.
func WithEnabled(enabled bool) Option {
	return func(s *Scheduler) { s.enabled = enabled }
}

// WithTimeout sets the maximum duration for a single scheduled run.
func WithTimeout(d time.Duration) Option {
	return func(s *Scheduler) { s.timeout = d }
}

// WithRepo sets the repository for recording failed import runs.
func WithRepo(repo *repository.Repository) Option {
	return func(s *Scheduler) { s.repo = repo }
}

// New creates a Scheduler with the given cron expression, file path, and run function.
// It validates the cron expression at construction time.
func New(cronExpr, filePath string, runFn func(context.Context, importer.ImportRequest) (*domain.ImportReport, error), opts ...Option) (*Scheduler, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid IMPORT_CRON %q: %w", cronExpr, err)
	}

	s := &Scheduler{
		cronExpr: cronExpr,
		filePath: filePath,
		runFn:    runFn,
		clock:    realClock{},
		enabled:  true,
		schedule: schedule,
		stopCh:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// Start launches the cron scheduler. The first run happens at the next scheduled time.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("scheduler: already running")
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	if !s.enabled {
		log.Println("scheduler: disabled via IMPORT_ENABLED")
		return
	}

	log.Printf("scheduler: started with cron=%s file=%s", s.cronExpr, s.filePath)

	go s.loop(ctx)
}

func (s *Scheduler) loop(ctx context.Context) {
	next := s.schedule.Next(s.clock.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-s.clock.After(next.Sub(s.clock.Now())):
			s.RunOnce(ctx)
			next = s.schedule.Next(s.clock.Now())
		}
	}
}

// RunOnce executes a single scheduled import synchronously.
// It is exported so tests can trigger runs without waiting for the cron cycle.
// Only one RunOnce may be active at a time; overlapping calls are silently skipped.
func (s *Scheduler) RunOnce(ctx context.Context) {
	if !s.inRun.CompareAndSwap(false, true) {
		return
	}
	defer s.inRun.Store(false)

	startTime := s.clock.Now()
	log.Printf("scheduler: job start trigger_type=scheduled source_file=%s cron_expression=%s timestamp=%s",
		s.filePath, s.cronExpr, startTime.Format(time.RFC3339))

	req := importer.ImportRequest{
		TriggerType: "scheduled",
		TriggeredBy: "cron",
		FileName:    s.filePath,
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		log.Printf("scheduler: job end status=failed duration=%s error=%q",
			time.Since(startTime), err)
		// Record a failed import run for audit trail if repo is available.
		if s.repo != nil {
			run := &domain.ImportRun{
				TriggerType:  "scheduled",
				TriggeredBy:  "cron",
				FileName:     s.filePath,
				Status:       "failed",
				RowsTotal:    0,
				StartedAt:    startTime,
				FinishedAt:   &startTime,
				ErrorMessage: err.Error(),
			}
			if tx, txErr := s.repo.BeginTx(ctx); txErr == nil {
				if createErr := s.repo.WithTx(tx).CreateImportRun(ctx, run); createErr == nil {
					_ = tx.Commit()
				} else {
					_ = tx.Rollback()
				}
			}
		}
		return
	}
	req.Data = data

	runCtx := ctx
	if s.timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	var report *domain.ImportReport
	var runErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				runErr = fmt.Errorf("panic: %v", r)
			}
		}()
		report, runErr = s.runFn(runCtx, req)
	}()

	endTime := s.clock.Now()
	duration := endTime.Sub(startTime)

	if runErr != nil {
		var inProgress *importer.ImportInProgressError
		if errors.As(runErr, &inProgress) {
			log.Printf("scheduler: job end status=skipped duration=%s reason=%q", duration, runErr.Error())
			return
		}
		log.Printf("scheduler: job end status=failed duration=%s error=%q", duration, runErr)
		return
	}

	status := "success"
	if len(report.Errors) > 0 {
		status = "partial"
	}
	log.Printf("scheduler: job end status=%s duration=%s rows_total=%d rows_created=%d rows_updated=%d rows_skipped=%d rows_failed=%d",
		status, duration, report.RowsTotal, report.Created, report.Updated, report.Skipped, len(report.Errors))
}

// Stop terminates the scheduler goroutine.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	log.Println("scheduler: stopped")
}
