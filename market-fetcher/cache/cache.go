package cache

import "time"

// LoaderFunc defines a function for loading data by missing keys.
// The function receives a list of keys that are missing from the cache,
// and should return a key->data map for those keys.
type LoaderFunc func(missingKeys []string) (map[string][]byte, error)

// Cache interface for universal multi-level cache
type Cache interface {
	// GetOrLoad retrieves data by keys from cache or loads them using LoaderFunc
	//
	// Parameters:
	// - keys: list of keys to retrieve data for
	// - loader: function to load missing data
	// - loadOnlyMissingKeys: if true, loader is called only with missing keys;
	//   if false, when any data is missing, loader is called with all keys
	// - ttl: time to live for cached data; if 0, uses cache's default expiration
	//
	// Returns:
	// - map[string][]byte: key->data map
	// - error: execution error
	GetOrLoad(keys []string, loader LoaderFunc, loadOnlyMissingKeys bool, ttl time.Duration) (map[string][]byte, error)

	// Get retrieves data by keys from cache
	//
	// Parameters:
	// - keys: list of keys to retrieve data for
	//
	// Returns:
	// - map[string][]byte: key->data map for found keys
	// - []string: list of missing keys
	// - error: execution error
	Get(keys []string) (map[string][]byte, []string, error)

	// Set stores data in cache with the specified TTL
	//
	// Parameters:
	// - data: key->data map to store in cache
	// - ttl: time to live for cached data; if 0, uses cache's default expiration
	//
	// Returns:
	// - error: execution error
	Set(data map[string][]byte, ttl time.Duration) error
}
