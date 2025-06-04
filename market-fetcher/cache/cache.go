package cache

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
	//
	// Returns:
	// - map[string][]byte: key->data map
	// - error: execution error
	GetOrLoad(keys []string, loader LoaderFunc, loadOnlyMissingKeys bool) (map[string][]byte, error)
}
