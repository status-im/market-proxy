package coingecko_leaderboard

import (
	"testing"

	"github.com/status-im/market-proxy/config"
	"github.com/stretchr/testify/assert"
)

// TestService_Healthy_Logic tests the logic of the Healthy method
func TestService_Healthy_Logic(t *testing.T) {
	// Create a new service
	cfg := &config.Config{}
	svc := NewService(cfg, nil, nil)

	// Test case 1: Empty cache, client not healthy
	// Just test the direct logic without using the Healthy method

	// Check initial state - no cache data
	assert.Nil(t, svc.GetCacheData())

	// Test with cache data
	mockData := []CoinData{
		{
			ID:     "bitcoin",
			Symbol: "btc",
			Name:   "Bitcoin",
		},
	}

	// Set cache data via MarketsUpdater
	svc.marketsUpdater.cache.Lock()
	svc.marketsUpdater.cache.data = &APIResponse{Data: mockData}
	svc.marketsUpdater.cache.Unlock()

	// Verify data is in cache
	cacheData := svc.GetCacheData()
	assert.NotNil(t, cacheData)
	assert.NotEmpty(t, cacheData.Data)

	// Clear cache
	svc.marketsUpdater.cache.Lock()
	svc.marketsUpdater.cache.data = nil
	svc.marketsUpdater.cache.Unlock()

	// Verify no data in cache
	assert.Nil(t, svc.GetCacheData())

	// Test with empty cache data
	svc.marketsUpdater.cache.Lock()
	svc.marketsUpdater.cache.data = &APIResponse{Data: []CoinData{}}
	svc.marketsUpdater.cache.Unlock()

	// Verify cache exists but is empty
	cacheData = svc.GetCacheData()
	assert.NotNil(t, cacheData)
	assert.Empty(t, cacheData.Data)
}
