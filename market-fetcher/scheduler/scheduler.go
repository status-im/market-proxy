package scheduler

import (
	"context"
	"sync"
	"time"
)

// Scheduler manages a background task that runs at regular intervals
type Scheduler struct {
	interval    time.Duration
	task        func(context.Context)
	wg          sync.WaitGroup
	mu          sync.Mutex
	running     bool
	cancel      context.CancelFunc
	taskRunning sync.Mutex // Mutex to prevent concurrent task executions
}

// New creates a new Scheduler instance
func New(interval time.Duration, task func(context.Context)) *Scheduler {
	return &Scheduler{
		interval: interval,
		task:     task,
	}
}

// Start begins executing the task at the specified interval
func (s *Scheduler) Start(ctx context.Context, firstRunImmediately bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	// Create a new context with cancellation
	ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// Execute the task immediately if requested
		if firstRunImmediately {
			s.runTaskIfNotRunning(ctx)
		}

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runTaskIfNotRunning(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// runTaskIfNotRunning runs the task only if another instance isn't already running
func (s *Scheduler) runTaskIfNotRunning(ctx context.Context) {
	// Try to acquire the mutex, but don't block if it's already locked
	if s.taskRunning.TryLock() {
		go func() {
			defer s.taskRunning.Unlock()
			s.task(ctx)
		}()
	}
}

// Stop terminates the periodic task execution
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.cancel != nil {
		s.cancel() // Cancel context to stop the goroutine
	}
	s.wg.Wait() // Wait for goroutine to complete
	s.running = false
}

// IsRunning returns true if the task is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
