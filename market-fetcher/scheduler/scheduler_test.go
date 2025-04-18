package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriodicTask(t *testing.T) {
	var counter int32

	// Create task that increments counter
	task := func(ctx context.Context) {
		atomic.AddInt32(&counter, 1)
	}

	// Create periodic task with 100ms interval
	pt := New(100*time.Millisecond, task)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the task
	pt.Start(ctx)
	assert.True(t, pt.IsRunning())

	// Wait for 3 executions
	time.Sleep(350 * time.Millisecond)

	// Stop the task
	pt.Stop()
	assert.False(t, pt.IsRunning())

	// Verify counter was incremented at least 3 times
	assert.GreaterOrEqual(t, atomic.LoadInt32(&counter), int32(3))

	// Wait a bit longer to ensure task is stopped
	time.Sleep(200 * time.Millisecond)
	finalCount := atomic.LoadInt32(&counter)

	// Verify counter didn't increment after stop
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, finalCount, atomic.LoadInt32(&counter))
}

func TestPeriodicTask_StopBeforeStart(t *testing.T) {
	pt := New(100*time.Millisecond, func(ctx context.Context) {})
	pt.Stop() // Should not panic
	assert.False(t, pt.IsRunning())
}

func TestPeriodicTask_DoubleStart(t *testing.T) {
	var counter int32
	pt := New(100*time.Millisecond, func(ctx context.Context) {
		atomic.AddInt32(&counter, 1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pt.Start(ctx)
	pt.Start(ctx) // Second start should be ignored

	time.Sleep(150 * time.Millisecond)
	pt.Stop()

	assert.GreaterOrEqual(t, atomic.LoadInt32(&counter), int32(1))
}

func TestPeriodicTask_ContextCancellation(t *testing.T) {
	var counter int32
	pt := New(100*time.Millisecond, func(ctx context.Context) {
		atomic.AddInt32(&counter, 1)
	})

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Start the task
	pt.Start(ctx)
	assert.True(t, pt.IsRunning())

	// Wait for at least one execution
	time.Sleep(150 * time.Millisecond)
	initialCount := atomic.LoadInt32(&counter)
	assert.Greater(t, initialCount, int32(0))

	// Cancel context and stop scheduler
	cancel()
	pt.Stop()

	// Wait for task to stop
	time.Sleep(200 * time.Millisecond)
	finalCount := atomic.LoadInt32(&counter)

	// Verify task stopped and counter didn't increment after cancellation
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, finalCount, atomic.LoadInt32(&counter))
	assert.False(t, pt.IsRunning())
}

func TestPeriodicTask_NestedContext(t *testing.T) {
	var counter int32
	pt := New(100*time.Millisecond, func(ctx context.Context) {
		atomic.AddInt32(&counter, 1)
	})

	// Create parent context with timeout
	parentCtx, parentCancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer parentCancel()

	// Create child context
	childCtx, childCancel := context.WithCancel(parentCtx)
	defer childCancel()

	// Start with child context
	pt.Start(childCtx)
	assert.True(t, pt.IsRunning())

	// Wait for parent context to timeout
	time.Sleep(400 * time.Millisecond)
	pt.Stop()

	// Verify task stopped due to parent context timeout
	assert.False(t, pt.IsRunning())
	assert.Greater(t, atomic.LoadInt32(&counter), int32(0))
}
