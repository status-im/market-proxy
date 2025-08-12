package events

//go:generate mockgen -destination=mocks/subscription.go . ISubscription,ISubscriptionManager

import (
	"context"
	"sync"
)

// ISubscription defines the contract for subscription objects
type ISubscription interface {
	// Chan returns a read-only channel for self-handling events
	Chan() <-chan struct{}
	// Cancel unsubscribes and closes the channel. Safe for repeated calls
	Cancel()
	// Watch starts a goroutine that calls cb on each event
	// If callNow is true, cb is called immediately
	// When parentCtx finishes, the subscription is automatically cancelled
	Watch(parentCtx context.Context, cb func(), callNow bool) ISubscription
}

// ISubscriptionManager defines the contract for managing subscriptions
type ISubscriptionManager interface {
	// Subscribe creates a new subscription and returns it
	Subscribe() ISubscription
	// Unsubscribe removes a subscription by its channel
	Unsubscribe(ch chan struct{})
	// Emit sends notification to all subscribers (non-blocking if their channel is full)
	Emit(ctx context.Context)
}

type Subscription struct {
	ch     chan struct{}
	mgr    *SubscriptionManager
	cancel context.CancelFunc
	once   sync.Once
}

// Chan returns a read-only channel for self-handling events.
func (s *Subscription) Chan() <-chan struct{} { return s.ch }

// Cancel unsubscribes and closes the channel. Safe for repeated calls.
func (s *Subscription) Cancel() {
	s.once.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		s.mgr.Unsubscribe(s.ch)
	})
}

// Watch starts a goroutine that calls cb on each event.
// If callNow is true, cb is called immediately.
// When parentCtx finishes, the subscription is automatically cancelled.
func (s *Subscription) Watch(parentCtx context.Context, cb func(), callNow bool) ISubscription {
	ctx, cancel := context.WithCancel(parentCtx)
	s.cancel = cancel

	if callNow {
		cb()
	}

	go func(ctx context.Context) {
		defer s.Cancel() // cancel subscription on exit
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.ch:
				cb()
			}
		}
	}(ctx)

	return s
}

type SubscriptionManager struct {
	mu          sync.RWMutex
	subscribers map[chan struct{}]struct{}
}

func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscribers: make(map[chan struct{}]struct{}),
	}
}

func (m *SubscriptionManager) Subscribe() ISubscription {
	ch := make(chan struct{}, 1)

	m.mu.Lock()
	m.subscribers[ch] = struct{}{}
	m.mu.Unlock()

	return &Subscription{ch: ch, mgr: m}
}

func (m *SubscriptionManager) Unsubscribe(ch chan struct{}) {
	m.mu.Lock()
	if _, ok := m.subscribers[ch]; ok {
		delete(m.subscribers, ch)
		close(ch)
	}
	m.mu.Unlock()
}

// Emit sends notification to all subscribers (non-blocking if their channel is full).
func (m *SubscriptionManager) Emit(ctx context.Context) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for sub := range m.subscribers {
		select {
		case <-ctx.Done():
			// Stop sending notifications when the context is cancelled
			return
		case sub <- struct{}{}:
			// Notified successfully
		default:
			// Skip notification if the subscriber's channel is full (non-blocking)
		}
	}
}
