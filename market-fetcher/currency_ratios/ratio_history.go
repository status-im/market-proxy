package currency_ratios

import (
	"sync"
	"time"
)

const (
	// RatioHistoryRetention is how far back snapshots are kept. It only needs to
	// cover the longest window we convert percent changes over (1h) plus slack,
	// since history is realtime-only and is never persisted.
	RatioHistoryRetention = 70 * time.Minute

	// maxRatioHistoryEntries bounds memory for pathologically short intervals
	maxRatioHistoryEntries = 4096

	// minRatioHistoryEntries keeps at least a current and a previous snapshot
	minRatioHistoryEntries = 2
)

// historyCapacity is how many snapshots must be retained to cover
// RatioHistoryRetention at the configured update interval.
func historyCapacity(interval time.Duration) int {
	if interval <= 0 {
		return minRatioHistoryEntries
	}

	capacity := int(RatioHistoryRetention/interval) + 2
	if capacity < minRatioHistoryEntries {
		capacity = minRatioHistoryEntries
	}
	if capacity > maxRatioHistoryEntries {
		capacity = maxRatioHistoryEntries
	}

	return capacity
}

// ratioHistory is a fixed-capacity ring of recent snapshots. Once full, adding a
// snapshot overwrites the oldest one.
type ratioHistory struct {
	mu      sync.RWMutex
	entries []*Snapshot
	next    int
	size    int
}

func newRatioHistory(capacity int) *ratioHistory {
	if capacity < 1 {
		capacity = 1
	}
	return &ratioHistory{entries: make([]*Snapshot, capacity)}
}

// add appends a snapshot, evicting the oldest one when the ring is full
func (h *ratioHistory) add(snapshot *Snapshot) {
	if snapshot == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries[h.next] = snapshot
	h.next = (h.next + 1) % len(h.entries)
	if h.size < len(h.entries) {
		h.size++
	}
}

// len returns the number of retained snapshots
func (h *ratioHistory) len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.size
}

// oldest returns the oldest retained snapshot, or nil when empty
func (h *ratioHistory) oldest() *Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.size == 0 {
		return nil
	}

	// The oldest entry sits at the write position once the ring wrapped around
	index := h.next - h.size
	if index < 0 {
		index += len(h.entries)
	}

	return h.entries[index]
}

// closestTo returns the retained snapshot whose timestamp is nearest to t,
// or nil when the history is empty.
func (h *ratioHistory) closestTo(t time.Time) *Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var best *Snapshot
	var bestDelta time.Duration

	for _, snapshot := range h.entries {
		if snapshot == nil {
			continue
		}

		delta := snapshot.UpdatedAt.Sub(t)
		if delta < 0 {
			delta = -delta
		}

		if best == nil || delta < bestDelta {
			best, bestDelta = snapshot, delta
		}
	}

	return best
}
