// Package scheduler runs the import on a cron schedule via a goroutine.
package scheduler

import (
	"context"
	"log"
	"os"
	"sync"

	"awesomeProject5/internal/domain"
	"github.com/robfig/cron/v3"
)

// Scheduler periodically triggers imports using a cron expression.
type Scheduler struct {
	cronExpr string
	filePath string
	runFn    func(context.Context, string, []byte) (*domain.ImportReport, error)
	cron     *cron.Cron
	mu       sync.Mutex
	running  bool
}

// New creates a Scheduler with the given cron expression, file path, and run function.
func New(cronExpr, filePath string, runFn func(context.Context, string, []byte) (*domain.ImportReport, error)) *Scheduler {
	return &Scheduler{
		cronExpr: cronExpr,
		filePath: filePath,
		runFn:    runFn,
	}
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
	s.mu.Unlock()

	c := cron.New()
	s.cron = c

	_, err := c.AddFunc(s.cronExpr, func() {
		s.runImport(ctx)
	})
	if err != nil {
		log.Printf("scheduler: invalid cron expression %q: %v", s.cronExpr, err)
		return
	}

	c.Start()
	log.Printf("scheduler: started with cron=%s file=%s", s.cronExpr, s.filePath)
}

func (s *Scheduler) runImport(ctx context.Context) {
	// Prevent overlapping imports.
	if !s.mu.TryLock() {
		log.Println("scheduler: previous import still running, skipping this run")
		return
	}
	defer s.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("scheduler: panic recovered: %v", r)
			log.Println("scheduler: job completed with errors")
		}
	}()

	log.Printf("scheduler: job started, importing file=%s", s.filePath)

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		log.Printf("scheduler: failed to read file %s: %v", s.filePath, err)
		log.Println("scheduler: job completed with errors")
		return
	}

	report, err := s.runFn(ctx, s.filePath, data)
	if err != nil {
		log.Printf("scheduler: import failed: %v", err)
		log.Println("scheduler: job completed with errors")
		return
	}

	log.Printf("scheduler: import statistics - total=%d created=%d updated=%d skipped=%d errors=%d",
		report.RowsTotal, report.Created, report.Updated, report.Skipped, len(report.Errors))

	for _, e := range report.Errors {
		log.Printf("scheduler: row %d error: %s", e.Row, e.Reason)
	}

	log.Println("scheduler: job completed successfully")
}

// Stop terminates the scheduler goroutine.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	if s.cron != nil {
		s.cron.Stop()
	}
	log.Println("scheduler: stopped")
}
