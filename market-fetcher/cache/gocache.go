package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// GoCache simple in-memory cache implementation using go-cache
type GoCache struct {
	cache *cache.Cache
}

// NewGoCache creates a new GoCache instance
// defaultExpiration: default expiration time for items
// cleanupInterval: interval for cleaning up expired items
func NewGoCache(defaultExpiration, cleanupInterval time.Duration) *GoCache {
	return &GoCache{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// GetResult represents the result of a Get operation
type GetResult struct {
	Found       map[string][]byte // keys that were found in cache
	MissingKeys []string          // keys that were not found
}

// Get retrieves values for the given keys
// Returns found data and list of missing keys
func (gc *GoCache) Get(keys []string) GetResult {
	result := GetResult{
		Found:       make(map[string][]byte),
		MissingKeys: make([]string, 0),
	}

	for _, key := range keys {
		if value, found := gc.cache.Get(key); found {
			if data, ok := value.([]byte); ok {
				result.Found[key] = data
			} else {
				// If stored value is not []byte, add to missing
				result.MissingKeys = append(result.MissingKeys, key)
			}
		} else {
			result.MissingKeys = append(result.MissingKeys, key)
		}
	}

	return result
}

// Set stores key-value pairs with specified timeout
// If timeout is 0, uses cache's default expiration
// If timeout is -1 (cache.NoExpiration), item never expires
func (gc *GoCache) Set(data map[string][]byte, timeout time.Duration) {
	for key, value := range data {
		gc.cache.Set(key, value, timeout)
	}
}

// Delete removes items from cache by keys
func (gc *GoCache) Delete(keys []string) {
	for _, key := range keys {
		gc.cache.Delete(key)
	}
}

// Clear removes all items from cache
func (gc *GoCache) Clear() {
	gc.cache.Flush()
}

// ItemCount returns the number of items in cache
func (gc *GoCache) ItemCount() int {
	return gc.cache.ItemCount()
}

// DeleteExpired manually triggers deletion of expired items
func (gc *GoCache) DeleteExpired() {
	gc.cache.DeleteExpired()
}
