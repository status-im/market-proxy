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
	// If the expiration duration is less than one (or NoExpiration),
	// the items in the cache never expire (by default)
	DefaultExpiration time.Duration `yaml:"default_expiration"`

	// CleanupInterval interval for cleaning up expired items
	// If the cleanup interval is less than one, expired items are not
	// deleted from the cache before calling c.DeleteExpired()
	CleanupInterval time.Duration `yaml:"cleanup_interval"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() Config {
	return Config{
		GoCache: GoCacheConfig{
			DefaultExpiration: 5 * time.Minute,
			CleanupInterval:   10 * time.Minute,
		},
	}
}
