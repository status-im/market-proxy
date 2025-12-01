package fetcher_by_id

import (
	"context"
	"time"

	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
)

// CachedData represents cached data with timestamp for staleness checks
type CachedData struct {
	// Data is the raw JSON response from the API
	Data []byte
	// Timestamp is when the data was fetched
	Timestamp time.Time
}

// IsExpired checks if the cached data has exceeded the given TTL
func (c *CachedData) IsExpired(ttl time.Duration) bool {
	return time.Since(c.Timestamp) > ttl
}

// IIdsProvider defines the interface for getting IDs to fetch
type IIdsProvider interface {
	// GetIds returns a list of IDs to fetch, limited to the specified count
	GetIds(limit int) ([]string, error)
}

// IGenericFetcher defines the interface for generic data fetching
type IGenericFetcher interface {
	// FetchSingle fetches data for a single ID
	// Returns the raw JSON response
	FetchSingle(id string) ([]byte, error)

	// FetchBatch fetches data for multiple IDs in a single request
	// Returns a map of ID -> raw JSON response
	FetchBatch(ids []string) (map[string][]byte, error)

	// Healthy returns true if the fetcher has had at least one successful fetch
	Healthy() bool
}

// IGenericService defines the interface for a generic data service
type IGenericService interface {
	// Start starts the service
	Start(ctx context.Context) error

	// Stop stops the service
	Stop()

	// GetByID returns cached data for a specific ID
	GetByID(id string) ([]byte, interfaces.CacheStatus, error)

	// Healthy returns true if the service is operational
	Healthy() bool

	// SubscribeOnUpdate subscribes to data update notifications
	SubscribeOnUpdate() events.ISubscription
}

// UpdateCallback is called when data is updated
// ctx is the context, data is a map of ID -> raw JSON response
type UpdateCallback func(ctx context.Context, data map[string][]byte) error

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	// ID is the identifier of the fetched item
	ID string
	// Data is the raw JSON response
	Data []byte
	// Error is set if the fetch failed
	Error error
}
