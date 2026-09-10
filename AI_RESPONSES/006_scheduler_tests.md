# Scheduler Tests Fix

**Date:** 2026-09-10 (Asia/Almaty)
**Mode:** 💻 Code
**Basis:** [AI_RESPONSES/005_api_layer.md](../005_api_layer.md)

## Overview

This document describes the fixes applied to the scheduler package to resolve failing tests. The scheduler package (`internal/scheduler`) implements cron-based scheduled imports with overlap prevention and panic recovery.

## Issues Found

Running `go test ./...` revealed 9 failing tests in `internal/scheduler`:

1. `TestScheduler_ValidCronFiresAtExpectedTimes` — expected 2 runs, got 1
2. `TestScheduler_RunOnce_OverlapSkip` — expected 1 run due to overlap, got 2
3. `TestScheduler_StartStop` — `invalid IMPORT_CRON "@every 1s": parser does not accept descriptors`
4. `TestScheduler_OverlapPrevention` — `invalid IMPORT_CRON "@every 500ms": parser does not accept descriptors`
5. `TestScheduler_ErrorHandling` — `invalid IMPORT_CRON "@every 500ms": parser does not accept descriptors`
6. `TestScheduler_PanicRecovery` — `invalid IMPORT_CRON "@every 500ms": parser does not accept descriptors`
7. `TestScheduler_FileReadError` — `invalid IMPORT_CRON "@every 1s": parser does not accept descriptors`
8. `TestScheduler_StatisticsLogging` — `invalid IMPORT_CRON "@every 1s": parser does not accept descriptors`
9. `TestScheduler_MultipleRuns` — `invalid IMPORT_CRON "@every 1s": parser does not accept descriptors`

## Root Causes

1. **Cron parser missing `Descriptor` option**: The `cron.NewParser()` call in `scheduler.go` only included standard field options (`Minute | Hour | Dom | Month | Dow`) but not `Descriptor`. This prevented parsing of `@every` style cron expressions used in several tests.

2. **No overlap prevention in `RunOnce`**: The `RunOnce` method had no guard against concurrent execution. When called while another run was in progress, it would execute the run function again instead of being skipped.

3. **No panic recovery**: If the run function panicked, the panic would propagate up and crash the scheduler goroutine, preventing future scheduled runs.

4. **Test race condition**: In `TestScheduler_ValidCronFiresAtExpectedTimes`, the test advanced the mock clock immediately after `sched.Start(ctx)`, but the scheduler loop goroutine might not have started yet. This caused the first scheduled time to be missed.

## Fixes Applied

### 1. Cron Parser — `@every` descriptor support

**File:** `internal/scheduler/scheduler.go`

```go
// Before:
parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// After:
parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
```

Adding `cron.Descriptor` enables parsing of `@every` expressions like `@every 1s` and `@every 500ms`.

### 2. Overlap Prevention in `RunOnce`

**File:** `internal/scheduler/scheduler.go`

Added an `atomic.Bool` field `inRun` to the `Scheduler` struct:

```go
type Scheduler struct {
    // ... existing fields ...
    inRun atomic.Bool
}
```

Modified `RunOnce` to use `CompareAndSwap`:

```go
func (s *Scheduler) RunOnce(ctx context.Context) {
    if !s.inRun.CompareAndSwap(false, true) {
        return  // Another run is already in progress, skip
    }
    defer s.inRun.Store(false)
    // ... rest of method
}
```

This ensures only one `RunOnce` execution is active at a time. Overlapping calls are silently skipped.

### 3. Panic Recovery

**File:** `internal/scheduler/scheduler.go`

Wrapped the `s.runFn` call in a `defer recover()`:

```go
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
```

Panics in the run function are now caught, converted to errors, and logged. The scheduler continues running.

### 4. Test Race Condition Fix

**File:** `internal/scheduler/scheduler_test.go`

Added a small sleep after `sched.Start(ctx)` to ensure the scheduler loop goroutine has started:

```go
sched.Start(ctx)
defer sched.Stop()

// Wait for the scheduler loop goroutine to start.
time.Sleep(50 * time.Millisecond)

// First fire at 1:00.
clock.Advance(1 * time.Hour)
```

## Validation

After applying all fixes:

```
$ go test ./...
ok  	awesomeProject5/internal/api	        (cached)
ok  	awesomeProject5/internal/importer	(cached)
ok  	awesomeProject5/internal/parser	        (cached)
ok  	awesomeProject5/internal/repository	(cached)
ok  	awesomeProject5/internal/scheduler	15.805s
```

All 9 previously failing scheduler tests now pass. The full test suite passes with no failures.

## Files Modified

- `internal/scheduler/scheduler.go` — cron parser, overlap prevention, panic recovery
- `internal/scheduler/scheduler_test.go` — race condition fix
