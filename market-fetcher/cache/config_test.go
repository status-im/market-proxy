package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	// Test GoCache defaults
	assert.True(t, config.GoCache.Enabled)
	assert.Equal(t, 5*time.Minute, config.GoCache.DefaultExpiration)
	assert.Equal(t, 10*time.Minute, config.GoCache.CleanupInterval)

	// Test Redis defaults
	assert.False(t, config.Redis.Enabled)
	assert.Equal(t, "localhost:6379", config.Redis.Address)
	assert.Equal(t, "", config.Redis.Password)
	assert.Equal(t, 0, config.Redis.Database)
	assert.Equal(t, 10*time.Minute, config.Redis.DefaultExpiration)
	assert.Equal(t, "market-proxy:", config.Redis.KeyPrefix)
}

func TestConfigYAMLMarshaling(t *testing.T) {
	config := DefaultCacheConfig()

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	assert.NoError(t, err)

	// Unmarshal back
	var unmarshaledConfig Config
	err = yaml.Unmarshal(data, &unmarshaledConfig)
	assert.NoError(t, err)

	// Compare
	assert.Equal(t, config.GoCache.Enabled, unmarshaledConfig.GoCache.Enabled)
	assert.Equal(t, config.GoCache.DefaultExpiration, unmarshaledConfig.GoCache.DefaultExpiration)
	assert.Equal(t, config.GoCache.CleanupInterval, unmarshaledConfig.GoCache.CleanupInterval)
	assert.Equal(t, config.Redis.Enabled, unmarshaledConfig.Redis.Enabled)
	assert.Equal(t, config.Redis.Address, unmarshaledConfig.Redis.Address)
}

func TestCustomCacheConfig(t *testing.T) {
	yamlConfig := `
go_cache:
  enabled: true
  default_expiration: 2m
  cleanup_interval: 5m
redis:
  enabled: true
  address: "redis.example.com:6379"
  password: "secret"
  database: 1
  default_expiration: 15m
  key_prefix: "test:"
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlConfig), &config)
	assert.NoError(t, err)

	// Test parsed values
	assert.True(t, config.GoCache.Enabled)
	assert.Equal(t, 2*time.Minute, config.GoCache.DefaultExpiration)
	assert.Equal(t, 5*time.Minute, config.GoCache.CleanupInterval)

	assert.True(t, config.Redis.Enabled)
	assert.Equal(t, "redis.example.com:6379", config.Redis.Address)
	assert.Equal(t, "secret", config.Redis.Password)
	assert.Equal(t, 1, config.Redis.Database)
	assert.Equal(t, 15*time.Minute, config.Redis.DefaultExpiration)
	assert.Equal(t, "test:", config.Redis.KeyPrefix)
}

func TestGoCacheWithConfig(t *testing.T) {
	config := DefaultCacheConfig()

	// Create cache with config
	cache := NewGoCache(config.GoCache.DefaultExpiration, config.GoCache.CleanupInterval)

	// Test that cache works with config values
	testData := map[string][]byte{
		"test": []byte("value"),
	}

	cache.Set(testData, 0) // Use default expiration from config

	result := cache.Get([]string{"test"})
	assert.Len(t, result.Found, 1)
	assert.Equal(t, []byte("value"), result.Found["test"])
	assert.Len(t, result.MissingKeys, 0)
}
