package fetcher_by_id

import (
	"context"
	"testing"
	"time"

	cache_mocks "github.com/status-im/market-proxy/cache/mocks"
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func createTestGenericConfig() *config.FetcherByIdConfig {
	return &config.FetcherByIdConfig{
		Name:           "test",
		EndpointPath:   "/api/v3/coins/{{id}}",
		TTL:            1 * time.Hour,
		UpdateInterval: 30 * time.Minute,
		TopIdsLimit:    100,
		ParamsOverride: map[string]interface{}{
			"localization": false,
		},
	}
}

func createTestGlobalConfig() *config.Config {
	return &config.Config{
		APITokens: &config.APITokens{
			Tokens: []string{"test-token"},
		},
	}
}

// MockIdsProvider implements IIdsProvider for testing
type MockIdsProvider struct {
	ids []string
	err error
}

func (m *MockIdsProvider) GetIds(limit int) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	if limit > 0 && limit < len(m.ids) {
		return m.ids[:limit], nil
	}
	return m.ids, nil
}

func TestService_NewService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	assert.NotNil(t, service)
	assert.Equal(t, "test", service.GetName())
	assert.Equal(t, fetcherCfg, service.GetConfig())
}

func TestService_StartWithoutCache(t *testing.T) {
	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, nil)

	err := service.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache dependency not provided")
}

func TestService_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	mockCache.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)
	service.SetIdsProvider(&MockIdsProvider{ids: []string{"bitcoin", "ethereum"}})

	err := service.Start(context.Background())
	assert.NoError(t, err)

	// Stop should not panic
	assert.NotPanics(t, func() {
		service.Stop()
	})
}

func TestService_GetByID_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	// Setup cache to return data
	cacheKey := "test:id:bitcoin"
	cachedData := map[string][]byte{
		cacheKey: []byte(`{"name": "Bitcoin"}`),
	}
	mockCache.EXPECT().Get([]string{cacheKey}).Return(cachedData, []string{}, nil)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	data, status, err := service.GetByID("bitcoin")

	assert.NoError(t, err)
	assert.Equal(t, interfaces.CacheStatusFull, status)
	assert.Equal(t, []byte(`{"name": "Bitcoin"}`), data)
}

func TestService_GetByID_CacheMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	// Setup cache to return miss
	cacheKey := "test:id:bitcoin"
	mockCache.EXPECT().Get([]string{cacheKey}).Return(map[string][]byte{}, []string{cacheKey}, nil)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	data, status, err := service.GetByID("bitcoin")

	assert.Error(t, err)
	assert.Equal(t, interfaces.CacheStatusMiss, status)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "not found")
}

func TestService_GetMultiple_AllHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	// Setup cache to return all data
	cacheKeys := []string{"test:id:bitcoin", "test:id:ethereum"}
	cachedData := map[string][]byte{
		"test:id:bitcoin":  []byte(`{"name": "Bitcoin"}`),
		"test:id:ethereum": []byte(`{"name": "Ethereum"}`),
	}
	mockCache.EXPECT().Get(cacheKeys).Return(cachedData, []string{}, nil)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	result, missing, status := service.GetMultiple([]string{"bitcoin", "ethereum"})

	assert.Equal(t, interfaces.CacheStatusFull, status)
	assert.Len(t, result, 2)
	assert.Len(t, missing, 0)
	assert.Contains(t, result, "bitcoin")
	assert.Contains(t, result, "ethereum")
}

func TestService_GetMultiple_PartialHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	// Setup cache to return partial data
	cacheKeys := []string{"test:id:bitcoin", "test:id:ethereum"}
	cachedData := map[string][]byte{
		"test:id:bitcoin": []byte(`{"name": "Bitcoin"}`),
		// ethereum is missing
	}
	missingKeys := []string{"test:id:ethereum"}
	mockCache.EXPECT().Get(cacheKeys).Return(cachedData, missingKeys, nil)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	result, missing, status := service.GetMultiple([]string{"bitcoin", "ethereum"})

	assert.Equal(t, interfaces.CacheStatusPartial, status)
	assert.Len(t, result, 1)
	assert.Len(t, missing, 1)
	assert.Contains(t, result, "bitcoin")
	assert.Contains(t, missing, "ethereum")
}

func TestService_GetMultiple_AllMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	// Setup cache to return no data
	cacheKeys := []string{"test:id:bitcoin", "test:id:ethereum"}
	mockCache.EXPECT().Get(cacheKeys).Return(map[string][]byte{}, cacheKeys, nil)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	result, missing, status := service.GetMultiple([]string{"bitcoin", "ethereum"})

	assert.Equal(t, interfaces.CacheStatusMiss, status)
	assert.Len(t, result, 0)
	assert.Len(t, missing, 2)
}

func TestService_GetMultiple_EmptyIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)
	// No cache calls expected for empty IDs

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	result, missing, status := service.GetMultiple([]string{})

	assert.Equal(t, interfaces.CacheStatusFull, status)
	assert.Len(t, result, 0)
	assert.Len(t, missing, 0)
}

func TestService_CacheByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	// Expect cache.Set to be called with prefixed keys
	expectedData := map[string][]byte{
		"test:id:bitcoin":  []byte(`{"name": "Bitcoin"}`),
		"test:id:ethereum": []byte(`{"name": "Ethereum"}`),
	}
	mockCache.EXPECT().Set(expectedData, 1*time.Hour).Return(nil)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	// Call the internal cacheByID method via onDataUpdated
	data := map[string][]byte{
		"bitcoin":  []byte(`{"name": "Bitcoin"}`),
		"ethereum": []byte(`{"name": "Ethereum"}`),
	}
	err := service.onDataUpdated(context.Background(), data)

	assert.NoError(t, err)
}

func TestService_Healthy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	// Before initialization, should not be healthy
	assert.False(t, service.Healthy())
}

func TestService_SubscribeOnUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	subscription := service.SubscribeOnUpdate()
	assert.NotNil(t, subscription)
}

func TestService_SetIdsProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	provider := &MockIdsProvider{ids: []string{"bitcoin", "ethereum"}}

	// Should not panic
	assert.NotPanics(t, func() {
		service.SetIdsProvider(provider)
	})
}

func TestService_SetExtraIdsProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := cache_mocks.NewMockICache(ctrl)

	globalCfg := createTestGlobalConfig()
	fetcherCfg := createTestGenericConfig()

	service := NewService(globalCfg, fetcherCfg, mockCache)

	provider := &MockIdsProvider{ids: []string{"token1", "token2"}}

	// Should not panic
	assert.NotPanics(t, func() {
		service.SetExtraIdsProvider(provider)
	})
}
