// Package scheduler runs the import on a fixed interval via a ticker goroutine.
package scheduler

import (
	"log"
	"time"
)

// Scheduler periodically triggers the import.
type Scheduler struct {
	interval time.Duration
	stop     chan struct{}
}

// New creates a Scheduler with the given interval.
func New(interval time.Duration) *Scheduler {
	return &Scheduler{
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start launches the ticker goroutine. The first run happens after one interval.
func (s *Scheduler) Start(run func()) {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Println("scheduler: triggering import")
				run()
			case <-s.stop:
				return
			}
		}
	}()
}

// Stop terminates the scheduler goroutine.
func (s *Scheduler) Stop() {
	close(s.stop)
}
