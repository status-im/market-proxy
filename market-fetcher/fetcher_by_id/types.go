package fetcher_by_id

import (
	"context"
	"time"

	"github.com/status-im/market-proxy/events"
	"github.com/status-im/market-proxy/interfaces"
)

type CachedData struct {
	Data      []byte
	Timestamp time.Time
}

func (c *CachedData) IsExpired(ttl time.Duration) bool {
	return time.Since(c.Timestamp) > ttl
}

type IIdsProvider interface {
	GetIds(limit int) ([]string, error)
}

type IGenericFetcher interface {
	FetchSingle(id string) ([]byte, error)
	FetchBatch(ids []string) (map[string][]byte, error)
	Healthy() bool
}

type IGenericService interface {
	Start(ctx context.Context) error
	Stop()
	GetByID(id string) ([]byte, interfaces.CacheStatus, error)
	Healthy() bool
	SubscribeOnUpdate() events.ISubscription
}

type UpdateCallback func(ctx context.Context, data map[string][]byte) error

type FetchResult struct {
	ID    string
	Data  []byte
	Error error
}
