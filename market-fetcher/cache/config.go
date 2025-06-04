package cache

import "time"

// Config represents cache configuration
type Config struct {
	// GoCache configuration
	GoCache GoCacheConfig `yaml:"go_cache"`

	// Redis configuration (for future use)
	Redis RedisConfig `yaml:"redis"`
}

// GoCacheConfig configuration for in-memory go-cache
type GoCacheConfig struct {
	// DefaultExpiration default expiration time for cache items
	// If 0, items never expire by default
	DefaultExpiration time.Duration `yaml:"default_expiration"`

	// CleanupInterval interval for cleaning up expired items
	// Should be less than DefaultExpiration
	CleanupInterval time.Duration `yaml:"cleanup_interval"`

	// Enabled whether go-cache is enabled
	Enabled bool `yaml:"enabled"`
}

// RedisConfig configuration for Redis cache (placeholder for future implementation)
type RedisConfig struct {
	// Enabled whether Redis cache is enabled
	Enabled bool `yaml:"enabled"`

	// Address Redis server address (host:port)
	Address string `yaml:"address"`

	// Password Redis password (optional)
	Password string `yaml:"password"`

	// Database Redis database number
	Database int `yaml:"database"`

	// DefaultExpiration default expiration time for cache items in Redis
	DefaultExpiration time.Duration `yaml:"default_expiration"`

	// KeyPrefix prefix for all cache keys in Redis
	KeyPrefix string `yaml:"key_prefix"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() Config {
	return Config{
		GoCache: GoCacheConfig{
			DefaultExpiration: 5 * time.Minute,
			CleanupInterval:   10 * time.Minute,
			Enabled:           true,
		},
		Redis: RedisConfig{
			Enabled:           false,
			Address:           "localhost:6379",
			Password:          "",
			Database:          0,
			DefaultExpiration: 10 * time.Minute,
			KeyPrefix:         "market-proxy:",
		},
	}
}
