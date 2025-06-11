package cache

import (
	"context"
	"fmt"
	"time"
)

// Service implements Cache interface with go-cache only
type Service struct {
	goCache *GoCache
	config  Config
}

// NewService creates a new cache service with the given configuration
func NewService(config Config) *Service {
	var goCache *GoCache

	// Create go-cache (L1 cache)
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

// Start implements core.Interface
func (s *Service) Start(ctx context.Context) error {
	// Cache service doesn't need startup logic
	// Just validate configuration
	if s.goCache == nil {
		return fmt.Errorf("cache service not properly initialized")
	}
	return nil
}

// Stop implements core.Interface
func (s *Service) Stop() {
	// Clear and close caches
	if s.goCache != nil {
		s.goCache.Clear()
	}
}

// GetOrLoad retrieves data by keys from local cache or loads them using LoaderFunc
func (s *Service) GetOrLoad(keys []string, loader LoaderFunc, loadOnlyMissingKeys bool, ttl time.Duration) (map[string][]byte, error) {
	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Step 1: Get from local cache
	result, missingKeys := s.getFromLocalCache(keys)

	// Step 2: Load missing data if needed
	if len(missingKeys) > 0 {
		keysToLoad := s.determineKeysToLoad(keys, missingKeys, loadOnlyMissingKeys)
		loadedData, err := s.loadAndCacheLocal(keysToLoad, loader, ttl)
		if err != nil {
			return nil, err
		}
		s.mergeResults(result, loadedData)
	}

	// Step 3: Prepare final result
	return s.prepareFinalResult(keys, result, loadOnlyMissingKeys), nil
}

// getFromLocalCache retrieves data from go-cache only
func (s *Service) getFromLocalCache(keys []string) (map[string][]byte, []string) {
	l1Result := s.goCache.Get(keys)
	return l1Result.Found, l1Result.MissingKeys
}

// loadAndCacheLocal loads data using loader function and updates local cache only
func (s *Service) loadAndCacheLocal(keysToLoad []string, loader LoaderFunc, ttl time.Duration) (map[string][]byte, error) {
	loadedData, err := loader(keysToLoad)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	// Update local cache with loaded data
	if len(loadedData) > 0 {
		s.goCache.Set(loadedData, ttl)
	}

	return loadedData, nil
}

// mergeResults merges source map into destination map
func (s *Service) mergeResults(dest, src map[string][]byte) {
	for key, value := range src {
		dest[key] = value
	}
}

// determineKeysToLoad decides which keys to load based on loadOnlyMissingKeys parameter
func (s *Service) determineKeysToLoad(originalKeys, missingKeys []string, loadOnlyMissingKeys bool) []string {
	if loadOnlyMissingKeys {
		return missingKeys
	}
	return originalKeys
}

// prepareFinalResult creates the final result map based on loadOnlyMissingKeys parameter
func (s *Service) prepareFinalResult(originalKeys []string, cachedData map[string][]byte, loadOnlyMissingKeys bool) map[string][]byte {
	if loadOnlyMissingKeys {
		// Return all cached data (includes both cached and loaded data)
		return cachedData
	}

	// Ensure all original keys are present in result
	result := make(map[string][]byte)
	for _, key := range originalKeys {
		if value, exists := cachedData[key]; exists {
			result[key] = value
		}
	}
	return result
}

// Stats returns statistics about the cache service
func (s *Service) Stats() ServiceStats {
	return ServiceStats{
		GoCacheItems: s.goCache.ItemCount(),
		Enabled:      s.config.GoCache.Enabled,
	}
}

// ServiceStats represents cache service statistics
type ServiceStats struct {
	GoCacheItems int  // Number of items in go-cache
	Enabled      bool // Whether go-cache is enabled
}

// Delete removes items from cache by keys
func (s *Service) Delete(keys []string) {
	if s.goCache != nil {
		s.goCache.Delete(keys)
	}
}

// Clear removes all items from cache
func (s *Service) Clear() {
	if s.goCache != nil {
		s.goCache.Clear()
	}
}
