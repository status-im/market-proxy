package config

import (
	"time"
)

type CoingeckoMarketsFetcher struct {
	ChunkSize    int           `yaml:"chunk_size"`    // Maximum items per request (250 for CoinGecko)
	RequestDelay time.Duration `yaml:"request_delay"` // Delay between requests
	TTL          time.Duration `yaml:"ttl"`           // Time to live for cached market data
}
