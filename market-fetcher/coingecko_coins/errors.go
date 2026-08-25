package coingecko_coins

import "github.com/status-im/market-proxy/fetcher_by_id"

// ErrNotFound is returned when no cached data exists for a coin id.
//
// It is the fetcher_by_id sentinel re-exported: callers of this package should
// not have to know which generic fetcher backs it, and errors.Is matches either
// name because they are the same value.
var ErrNotFound = fetcher_by_id.ErrNotFound
