package coingecko_coins

import (
	"testing"
	"time"

	cache_mocks "github.com/status-im/market-proxy/cache/mocks"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
	mock_interfaces "github.com/status-im/market-proxy/interfaces/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func createTestConfig() *config.Config {
	return &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
		CoingeckoCoins: config.FetcherByIdConfig{
			Name:         "coins",
			EndpointPath: "/api/v3/coins/{{id}}",
			TTL:          72 * time.Hour,
			ParamsOverride: map[string]interface{}{
				"localization": false,
				"tickers":      false,
			},
			Tiers: []config.GenericTier{
				{
					Name:           "top-100",
					IdFrom:         1,
					IdTo:           100,
					UpdateInterval: 24 * time.Hour,
				},
			},
		},
	}
}

func TestNewService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	assert.NotNil(t, service)
	assert.NotNil(t, service.genericService)
	assert.NotNil(t, service.marketsService)
	assert.Equal(t, cfg, service.cfg)
}

func TestService_GetCoin_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)

	// Setup cache to return data
	cacheKey := "coins:id:bitcoin"
	cachedData := map[string][]byte{
		cacheKey: []byte(`{"id":"bitcoin","name":"Bitcoin","symbol":"btc"}`),
	}
	mockCache.EXPECT().Get([]string{cacheKey}).Return(cachedData, []string{}, nil)

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	data, status, err := service.GetCoin("bitcoin")

	assert.NoError(t, err)
	assert.Equal(t, interfaces.CacheStatusFull, status)
	assert.Contains(t, string(data), "bitcoin")
}

func TestService_GetCoin_CacheMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)

	// Setup cache to return miss
	cacheKey := "coins:id:unknown-coin"
	mockCache.EXPECT().Get([]string{cacheKey}).Return(map[string][]byte{}, []string{cacheKey}, nil)

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	data, status, err := service.GetCoin("unknown-coin")

	assert.Error(t, err)
	assert.Equal(t, interfaces.CacheStatusMiss, status)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_GetMultipleCoins_AllHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)

	// Setup cache to return all data
	cacheKeys := []string{"coins:id:bitcoin", "coins:id:ethereum"}
	cachedData := map[string][]byte{
		"coins:id:bitcoin":  []byte(`{"id":"bitcoin","name":"Bitcoin"}`),
		"coins:id:ethereum": []byte(`{"id":"ethereum","name":"Ethereum"}`),
	}
	mockCache.EXPECT().Get(cacheKeys).Return(cachedData, []string{}, nil)

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	result, missing, status := service.GetMultipleCoins([]string{"bitcoin", "ethereum"})

	assert.Equal(t, interfaces.CacheStatusFull, status)
	assert.Len(t, result, 2)
	assert.Len(t, missing, 0)
	assert.Contains(t, result, "bitcoin")
	assert.Contains(t, result, "ethereum")
}

func TestService_GetMultipleCoins_PartialHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)

	// Setup cache to return partial data
	cacheKeys := []string{"coins:id:bitcoin", "coins:id:ethereum"}
	cachedData := map[string][]byte{
		"coins:id:bitcoin": []byte(`{"id":"bitcoin","name":"Bitcoin"}`),
	}
	missingKeys := []string{"coins:id:ethereum"}
	mockCache.EXPECT().Get(cacheKeys).Return(cachedData, missingKeys, nil)

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	result, missing, status := service.GetMultipleCoins([]string{"bitcoin", "ethereum"})

	assert.Equal(t, interfaces.CacheStatusPartial, status)
	assert.Len(t, result, 1)
	assert.Len(t, missing, 1)
	assert.Contains(t, result, "bitcoin")
	assert.Contains(t, missing, "ethereum")
}

func TestService_GetMultipleCoins_EmptyIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)
	// No cache calls expected for empty IDs

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	result, missing, status := service.GetMultipleCoins([]string{})

	assert.Equal(t, interfaces.CacheStatusFull, status)
	assert.Len(t, result, 0)
	assert.Len(t, missing, 0)
}

func TestService_Healthy_NotInitialized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	// Before any data is fetched, service should not be healthy
	assert.False(t, service.Healthy())
}

func TestService_SubscribeOnCoinsUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)

	cfg := createTestConfig()
	service := NewService(cfg, mockMarkets, mockCache)

	subscription := service.SubscribeOnCoinsUpdate()
	assert.NotNil(t, subscription)
}

func TestMarketsIdsProvider_GetIds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMarkets := mock_interfaces.NewMockIMarketsService(ctrl)
	mockMarkets.EXPECT().TopMarketIds(100).Return([]string{"bitcoin", "ethereum", "solana"}, nil)

	provider := &marketsIdsProvider{marketsService: mockMarkets}

	ids, err := provider.GetIds(100)

	assert.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.Equal(t, "bitcoin", ids[0])
	assert.Equal(t, "ethereum", ids[1])
	assert.Equal(t, "solana", ids[2])
}
