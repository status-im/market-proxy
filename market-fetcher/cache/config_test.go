package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultCacheConfig()

	// Test go-cache defaults
	assert.Equal(t, 5*time.Minute, config.GoCache.DefaultExpiration)
	assert.Equal(t, 10*time.Minute, config.GoCache.CleanupInterval)
}

func TestConfig_YAMLSerialization(t *testing.T) {
	config := DefaultCacheConfig()

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	assert.NoError(t, err)

	// Unmarshal back
	var unmarshaledConfig Config
	err = yaml.Unmarshal(data, &unmarshaledConfig)
	assert.NoError(t, err)

	// Compare
	assert.Equal(t, config.GoCache.DefaultExpiration, unmarshaledConfig.GoCache.DefaultExpiration)
	assert.Equal(t, config.GoCache.CleanupInterval, unmarshaledConfig.GoCache.CleanupInterval)
}

func TestConfig_YAMLDeserialization(t *testing.T) {
	yamlData := `
go_cache:
  default_expiration: 15m
  cleanup_interval: 30m
`

	var config Config
	err := yaml.Unmarshal([]byte(yamlData), &config)
	assert.NoError(t, err)

	// Verify go-cache config
	assert.Equal(t, 15*time.Minute, config.GoCache.DefaultExpiration)
	assert.Equal(t, 30*time.Minute, config.GoCache.CleanupInterval)
}
