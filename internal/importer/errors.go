package importer

import (
	"errors"
	"fmt"
	"time"
)

// ErrImportInProgress is returned when an import is already running.
var ErrImportInProgress = errors.New("import already in progress")

// ImportInProgressError provides details about the currently running import.
type ImportInProgressError struct {
	StartedAt   time.Time
	TriggeredBy string
	TriggerType string
}

func (e *ImportInProgressError) Error() string {
	return fmt.Sprintf("skipped: import already in progress (started at %s, triggered by %s)",
		e.StartedAt.Format(time.RFC3339), e.TriggeredBy)
}

func (e *ImportInProgressError) Unwrap() error {
	return ErrImportInProgress
}
