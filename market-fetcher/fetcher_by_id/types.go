package fetcher_by_id

import (
	"context"
	"time"
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

type UpdateCallback func(ctx context.Context, data map[string][]byte) error

type FetchResult struct {
	ID    string
	Data  []byte
	Error error
}
