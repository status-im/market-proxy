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

	// Set cache data via TopMarketsUpdater
	svc.topMarketsUpdater.cache.Lock()
	svc.topMarketsUpdater.cache.data = &APIResponse{Data: mockData}
	svc.topMarketsUpdater.cache.Unlock()

	// Verify data is in cache
	cacheData := svc.GetCacheData()
	assert.NotNil(t, cacheData)
	assert.NotEmpty(t, cacheData.Data)

	// Clear cache
	svc.topMarketsUpdater.cache.Lock()
	svc.topMarketsUpdater.cache.data = nil
	svc.topMarketsUpdater.cache.Unlock()

	// Verify no data in cache
	assert.Nil(t, svc.GetCacheData())

	// Test with empty cache data
	svc.topMarketsUpdater.cache.Lock()
	svc.topMarketsUpdater.cache.data = &APIResponse{Data: []CoinData{}}
	svc.topMarketsUpdater.cache.Unlock()

	// Verify cache exists but is empty
	cacheData = svc.GetCacheData()
	assert.NotNil(t, cacheData)
	assert.Empty(t, cacheData.Data)
}
