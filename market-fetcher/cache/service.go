package cache

import (
	"fmt"
	"time"
)

// Service implements Cache interface with go-cache backend
type Service struct {
	goCache *GoCache
	config  Config
}

// NewService creates a new cache service with the given configuration
func NewService(config Config) *Service {
	var goCache *GoCache

	if config.GoCache.Enabled {
		goCache = NewGoCache(config.GoCache.DefaultExpiration, config.GoCache.CleanupInterval)
	} else {
		// Create a minimal cache even if disabled for consistency
		goCache = NewGoCache(1*time.Minute, 2*time.Minute)
	}

	return &Service{
		goCache: goCache,
		config:  config,
	}
}

// GetOrLoad retrieves data by keys from cache or loads them using LoaderFunc
// Implements the Cache interface
func (s *Service) GetOrLoad(keys []string, loader LoaderFunc, loadOnlyMissingKeys bool) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Step 1: Get data from go-cache
	result := s.goCache.Get(keys)

	// If all data found, return immediately (fastest path)
	if len(result.MissingKeys) == 0 {
		return result.Found, nil
	}

	// Step 2: Determine which keys to load
	var keysToLoad []string
	if loadOnlyMissingKeys {
		// Load only missing keys (saves bandwidth)
		keysToLoad = result.MissingKeys
	} else {
		// Load all keys (ensures consistency)
		keysToLoad = keys
	}

	// Step 3: Load missing data using LoaderFunc
	loadedData, err := loader(keysToLoad)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Step 4: Update cache with loaded data
	if len(loadedData) > 0 {
		s.goCache.Set(loadedData, 0) // Use default expiration
	}

	// Step 5: Merge results
	finalResult := make(map[string][]byte)

	if loadOnlyMissingKeys {
		// Merge found data from cache with newly loaded data
		for key, value := range result.Found {
			finalResult[key] = value
		}
		for key, value := range loadedData {
			finalResult[key] = value
		}
	} else {
		// Use only newly loaded data (it includes all requested keys)
		// But fallback to cached data for keys not returned by loader
		for key, value := range loadedData {
			finalResult[key] = value
		}
		// Add any keys that were in cache but not returned by loader
		for _, key := range keys {
			if _, exists := finalResult[key]; !exists {
				if cachedValue, exists := result.Found[key]; exists {
					finalResult[key] = cachedValue
				}
			}
		}
	}

	return finalResult, nil
}

// Stats returns basic statistics about the cache service
func (s *Service) Stats() ServiceStats {
	return ServiceStats{
		GoCacheItems: s.goCache.ItemCount(),
		Enabled:      s.config.GoCache.Enabled,
	}
}

// ServiceStats represents cache service statistics
type ServiceStats struct {
	GoCacheItems int  // Number of items in go-cache
	Enabled      bool // Whether cache is enabled
}

// Clear clears all data from the cache
func (s *Service) Clear() {
	s.goCache.Clear()
}

// Delete removes specific keys from the cache
func (s *Service) Delete(keys []string) {
	s.goCache.Delete(keys)
}
