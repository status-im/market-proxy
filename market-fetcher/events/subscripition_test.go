package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionManager(t *testing.T) {
	// Create a new SubscriptionManager
	sm := NewSubscriptionManager()

	// Create a context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Test adding subscribers and receiving notifications
	subscriberCount := 5
	notificationReceived := make([]bool, subscriberCount)

	for i := 0; i < subscriberCount; i++ {
		sub := sm.Subscribe()
		idx := i // Copy the value of i to use inside the goroutine

		wg.Add(1)
		go func(sub SubscriptionInterface, idx int) {
			defer wg.Done()
			select {
			case <-sub.Chan():
				notificationReceived[idx] = true
			case <-time.After(1 * time.Second):
				// Timeout waiting for notification
			}
		}(sub, idx)
	}

	// Emit a notification to all subscribers
	sm.Emit(ctx)

	// Wait for all goroutines to finish
	wg.Wait()

	// Verify that all subscribers received the notification
	for i, received := range notificationReceived {
		require.Truef(t, received, "Subscriber %d did not receive notification", i)
	}

	// Test that notifications are not sent to closed channels
	subClosed := sm.Subscribe()
	// Ensure that cancel handles already closed channels properly
	subClosed.Cancel()

	// Test double cancel is safe (should not panic)
	require.NotPanics(t, func() {
		subClosed.Cancel()
	}, "Double cancel should be safe")
	// Emit a notification
	sm.Emit(ctx)

	// If no panic occurs, the test is successful
}

func TestSubscriptionManager_MultipleEmitsCollapse(t *testing.T) {
	sm := NewSubscriptionManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := sm.Subscribe()
	defer sub.Cancel()

	var received int
	var mu sync.Mutex

	go func() {
		for range sub.Chan() {
			mu.Lock()
			received++
			mu.Unlock()
		}
	}()

	// Emit multiple notifications
	sm.Emit(ctx)
	sm.Emit(ctx)
	sm.Emit(ctx)

	// Allow some time for the goroutine to process notifications
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	require.Equalf(t, 1, received, "Expected 1 notifications, but received %d", received)
	mu.Unlock()
}
