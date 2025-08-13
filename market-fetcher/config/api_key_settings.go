package config

// APIKeyConfig configures rate limiting per CoinGecko key type
type APIKeyConfig struct {
	// Requests per minute and burst per type. If zero, defaults are used.
	Pro   RateLimit `yaml:"pro"`
	Demo  RateLimit `yaml:"demo"`
	NoKey RateLimit `yaml:"nokey"`
}

// RateLimit represents a simple rpm + burst pair
type RateLimit struct {
	RateLimitPerMinute int `yaml:"rate_limit_per_minute"`
	Burst              int `yaml:"burst"`
}
