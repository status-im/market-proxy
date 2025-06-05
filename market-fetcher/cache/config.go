package cache

import "time"

// Config represents cache configuration
type Config struct {
	// GoCache configuration
	GoCache GoCacheConfig `yaml:"go_cache"`
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

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() Config {
	return Config{
		GoCache: GoCacheConfig{
			DefaultExpiration: 5 * time.Minute,
			CleanupInterval:   10 * time.Minute,
			Enabled:           true,
		},
	}
}
