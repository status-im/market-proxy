package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoCache_Basic(t *testing.T) {
	// Create cache with 5 minute default expiration and 10 minute cleanup interval
	cache := NewGoCache(5*time.Minute, 10*time.Minute)

	// Test Set and Get
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}

	// Set data with default expiration (0 = use default)
	cache.Set(testData, 0)

	// Test Get with all existing keys
	result := cache.Get([]string{"key1", "key2", "key3"})
	assert.Len(t, result.Found, 3)
	assert.Len(t, result.MissingKeys, 0)
	assert.Equal(t, []byte("value1"), result.Found["key1"])
	assert.Equal(t, []byte("value2"), result.Found["key2"])
	assert.Equal(t, []byte("value3"), result.Found["key3"])

	// Test Get with mixed existing and missing keys
	result = cache.Get([]string{"key1", "missing1", "key3", "missing2"})
	assert.Len(t, result.Found, 2)
	assert.Len(t, result.MissingKeys, 2)
	assert.Equal(t, []byte("value1"), result.Found["key1"])
	assert.Equal(t, []byte("value3"), result.Found["key3"])
	assert.Contains(t, result.MissingKeys, "missing1")
	assert.Contains(t, result.MissingKeys, "missing2")

	// Test ItemCount
	assert.Equal(t, 3, cache.ItemCount())
}

func TestGoCache_Delete(t *testing.T) {
	cache := NewGoCache(5*time.Minute, 10*time.Minute)

	// Set test data
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
		"key3": []byte("value3"),
	}
	cache.Set(testData, 0)

	// Delete some keys
	cache.Delete([]string{"key1", "key3"})

	// Verify deletion
	result := cache.Get([]string{"key1", "key2", "key3"})
	assert.Len(t, result.Found, 1)
	assert.Len(t, result.MissingKeys, 2)
	assert.Equal(t, []byte("value2"), result.Found["key2"])
	assert.Contains(t, result.MissingKeys, "key1")
	assert.Contains(t, result.MissingKeys, "key3")

	assert.Equal(t, 1, cache.ItemCount())
}

func TestGoCache_Clear(t *testing.T) {
	cache := NewGoCache(5*time.Minute, 10*time.Minute)

	// Set test data
	testData := map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}
	cache.Set(testData, 0)

	assert.Equal(t, 2, cache.ItemCount())

	// Clear cache
	cache.Clear()

	// Verify all data is gone
	result := cache.Get([]string{"key1", "key2"})
	assert.Len(t, result.Found, 0)
	assert.Len(t, result.MissingKeys, 2)
	assert.Equal(t, 0, cache.ItemCount())
}

func TestGoCache_Expiration(t *testing.T) {
	cache := NewGoCache(5*time.Minute, 10*time.Minute)

	// Set data with short expiration
	testData := map[string][]byte{
		"short":   []byte("expires soon"),
		"forever": []byte("never expires"),
	}

	// Set with 100ms expiration
	cache.Set(map[string][]byte{"short": testData["short"]}, 100*time.Millisecond)
	// Set with no expiration
	cache.Set(map[string][]byte{"forever": testData["forever"]}, -1)

	// Both should be available immediately
	result := cache.Get([]string{"short", "forever"})
	assert.Len(t, result.Found, 2)
	assert.Len(t, result.MissingKeys, 0)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Force cleanup of expired items
	cache.DeleteExpired()

	// Only "forever" should remain
	result = cache.Get([]string{"short", "forever"})
	assert.Len(t, result.Found, 1)
	assert.Len(t, result.MissingKeys, 1)
	assert.Equal(t, []byte("never expires"), result.Found["forever"])
	assert.Contains(t, result.MissingKeys, "short")
}

func TestGoCache_EmptyKeys(t *testing.T) {
	cache := NewGoCache(5*time.Minute, 10*time.Minute)

	// Test Get with empty keys slice
	result := cache.Get([]string{})
	assert.Len(t, result.Found, 0)
	assert.Len(t, result.MissingKeys, 0)

	// Test Set with empty data
	cache.Set(map[string][]byte{}, 0)
	assert.Equal(t, 0, cache.ItemCount())

	// Test Delete with empty keys
	cache.Delete([]string{})
	assert.Equal(t, 0, cache.ItemCount())
}
