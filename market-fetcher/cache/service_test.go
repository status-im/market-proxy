package cache

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Basic(t *testing.T) {
	config := DefaultCacheConfig()
	service := NewService(config)

	// Mock loader function
	loader := func(missingKeys []string) (map[string][]byte, error) {
		result := make(map[string][]byte)
		for _, key := range missingKeys {
			result[key] = []byte("loaded_" + key)
		}
		return result, nil
	}

	// Test with empty cache - should call loader
	data, err := service.GetOrLoad([]string{"key1", "key2"}, loader, true, 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, data, 2)
	assert.Equal(t, []byte("loaded_key1"), data["key1"])
	assert.Equal(t, []byte("loaded_key2"), data["key2"])

	// Test cache hit - should not call loader
	loaderCallCount := 0
	countingLoader := func(missingKeys []string) (map[string][]byte, error) {
		loaderCallCount++
		return loader(missingKeys)
	}

	data, err = service.GetOrLoad([]string{"key1", "key2"}, countingLoader, true, 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, data, 2)
	assert.Equal(t, 0, loaderCallCount) // Loader should not be called
}

func TestService_PartialCacheHit(t *testing.T) {
	config := DefaultCacheConfig()
	service := NewService(config)

	// Pre-populate cache with some data
	initialData := map[string][]byte{
		"cached_key": []byte("cached_value"),
	}
	err := service.goCache.Set(initialData, 0)
	require.NoError(t, err)

	// Mock loader for missing keys
	loader := func(missingKeys []string) (map[string][]byte, error) {
		result := make(map[string][]byte)
		for _, key := range missingKeys {
			result[key] = []byte("loaded_" + key)
		}
		return result, nil
	}

	// Test with mixed cached and missing keys
	data, err := service.GetOrLoad([]string{"cached_key", "missing_key1", "missing_key2"}, loader, true, 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, data, 3)
	assert.Equal(t, []byte("cached_value"), data["cached_key"])
	assert.Equal(t, []byte("loaded_missing_key1"), data["missing_key1"])
	assert.Equal(t, []byte("loaded_missing_key2"), data["missing_key2"])

	// Verify cache now contains all data
	assert.Equal(t, 3, service.goCache.ItemCount())
}

func TestService_LoadOnlyMissingKeys(t *testing.T) {
	config := DefaultCacheConfig()
	service := NewService(config)

	// Pre-populate cache
	initialData := map[string][]byte{
		"key1": []byte("cached_value1"),
	}
	err := service.goCache.Set(initialData, 0)
	require.NoError(t, err)

	// Track which keys are requested from loader
	var requestedKeys []string
	loader := func(missingKeys []string) (map[string][]byte, error) {
		requestedKeys = missingKeys
		result := make(map[string][]byte)
		for _, key := range missingKeys {
			result[key] = []byte("loaded_" + key)
		}
		return result, nil
	}

	// Test loadOnlyMissingKeys = true
	data, err := service.GetOrLoad([]string{"key1", "key2", "key3"}, loader, true, 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, data, 3)
	assert.Equal(t, []string{"key2", "key3"}, requestedKeys) // Only missing keys requested

	// Create a new service for the second test to ensure cache miss
	service2 := NewService(config)
	err = service2.goCache.Set(map[string][]byte{"key1": []byte("cached_value1")}, 0)
	require.NoError(t, err)

	// Reset and test loadOnlyMissingKeys = false
	requestedKeys = nil
	loader2 := func(missingKeys []string) (map[string][]byte, error) {
		requestedKeys = missingKeys
		result := make(map[string][]byte)
		for _, key := range missingKeys {
			result[key] = []byte("loaded_" + key)
		}
		return result, nil
	}

	data, err = service2.GetOrLoad([]string{"key1", "key2", "key3"}, loader2, false, 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, data, 3)
	assert.Equal(t, []string{"key1", "key2", "key3"}, requestedKeys) // All keys requested
}

func TestService_LoaderError(t *testing.T) {
	config := DefaultCacheConfig()
	service := NewService(config)

	// Mock loader that returns error
	loader := func(missingKeys []string) (map[string][]byte, error) {
		return nil, errors.New("loader failed")
	}

	// Test error handling
	data, err := service.GetOrLoad([]string{"key1"}, loader, true, 5*time.Minute)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "failed to load data")
}

func TestService_EmptyKeys(t *testing.T) {
	config := DefaultCacheConfig()
	service := NewService(config)

	loader := func(missingKeys []string) (map[string][]byte, error) {
		t.Fatal("Loader should not be called for empty keys")
		return nil, nil
	}

	// Test with empty keys slice
	data, err := service.GetOrLoad([]string{}, loader, true, 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, data, 0)
}

func TestService_ClearAndDelete(t *testing.T) {
	config := DefaultCacheConfig()
	service := NewService(config)

	// Add some data
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}
	err := service.goCache.Set(testData, 0)
	require.NoError(t, err)

	assert.Equal(t, 3, service.goCache.ItemCount())

	// Test Delete
	service.Delete([]string{"key1", "key3"})
	assert.Equal(t, 1, service.goCache.ItemCount())

	// Test Clear
	service.Clear()
	assert.Equal(t, 0, service.goCache.ItemCount())
}

func TestService_Implementation(t *testing.T) {
	// Verify that Service implements Cache interface
	config := DefaultCacheConfig()
	var cache Cache = NewService(config)

	loader := func(missingKeys []string) (map[string][]byte, error) {
		return map[string][]byte{"test": []byte("value")}, nil
	}

	data, err := cache.GetOrLoad([]string{"test"}, loader, true, 5*time.Minute)
	assert.NoError(t, err)
	assert.Len(t, data, 1)
}
