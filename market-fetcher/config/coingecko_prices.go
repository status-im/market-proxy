package config

import "time"

// CoingeckoPricesFetcher represents configuration for CoinGecko prices service
type CoingeckoPricesFetcher struct {
	ChunkSize    int           `yaml:"chunk_size"`    // Number of tokens to fetch in one request
	RequestDelay time.Duration `yaml:"request_delay"` // Delay between requests
	Currencies   []string      `yaml:"currencies"`    // Default currencies to fetch
}
