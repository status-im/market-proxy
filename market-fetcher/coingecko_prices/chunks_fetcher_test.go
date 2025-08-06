package coingecko_prices

import (
	"context"
	"testing"
	"time"

	cg "github.com/status-im/market-proxy/interfaces"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIClient is a mock implementation of APIClient
type MockAPIClient struct {
	mock.Mock
}

// FetchPrices mocks the FetchPrices method
func (m *MockAPIClient) FetchPrices(params cg.PriceParams) (map[string][]byte, error) {
	args := m.Called(params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]byte), args.Error(1)
}

// Healthy mocks the Healthy method
func (m *MockAPIClient) Healthy() bool {
	args := m.Called()
	return args.Bool(0)
}

func TestNewChunksFetcher(t *testing.T) {
	mockClient := new(MockAPIClient)
	mockClient.On("Healthy").Return(true)

	fetcher := NewChunksFetcher(mockClient, 100, 1000)
	assert.NotNil(t, fetcher)
	assert.Equal(t, 100, fetcher.chunkSize)
	assert.Equal(t, 1000*time.Millisecond, fetcher.requestDelay)
}

func TestChunksFetcher_FetchPrices_Success(t *testing.T) {
	mockClient := new(MockAPIClient)
	mockClient.On("Healthy").Return(true)

	// Set up expectations for two chunks
	mockClient.On("FetchPrices", cg.PriceParams{
		IDs:        []string{"token1", "token2"},
		Currencies: []string{"usd"},
	}).Return(map[string][]byte{
		"token1": []byte(`{"usd": 1.0}`),
		"token2": []byte(`{"usd": 2.0}`),
	}, nil)

	mockClient.On("FetchPrices", cg.PriceParams{
		IDs:        []string{"token3"},
		Currencies: []string{"usd"},
	}).Return(map[string][]byte{
		"token3": []byte(`{"usd": 3.0}`),
	}, nil)

	fetcher := NewChunksFetcher(mockClient, 2, 0)
	params := cg.PriceParams{
		IDs:        []string{"token1", "token2", "token3"},
		Currencies: []string{"usd"},
	}
	tokenData, err := fetcher.FetchPrices(context.Background(), params, nil)

	assert.NoError(t, err)
	assert.NotNil(t, tokenData)
	assert.Equal(t, 3, len(tokenData)) // 3 tokens
	assert.Contains(t, tokenData, "token1")
	assert.Contains(t, tokenData, "token2")
	assert.Contains(t, tokenData, "token3")
}

func TestChunksFetcher_FetchPrices_Error(t *testing.T) {
	mockClient := new(MockAPIClient)
	mockClient.On("Healthy").Return(true)

	// Set up expectation for error
	mockClient.On("FetchPrices", cg.PriceParams{
		IDs:        []string{"token1", "token2"},
		Currencies: []string{"usd"},
	}).Return(nil, assert.AnError)

	fetcher := NewChunksFetcher(mockClient, 2, 0)
	params := cg.PriceParams{
		IDs:        []string{"token1", "token2"},
		Currencies: []string{"usd"},
	}
	tokenData, err := fetcher.FetchPrices(context.Background(), params, nil)

	assert.Error(t, err)
	assert.Nil(t, tokenData)
}

func TestChunksFetcher_FetchPrices_EmptyInput(t *testing.T) {
	mockClient := new(MockAPIClient)
	mockClient.On("Healthy").Return(true)

	fetcher := NewChunksFetcher(mockClient, 2, 0)
	params := cg.PriceParams{
		IDs:        []string{},
		Currencies: []string{"usd"},
	}
	tokenData, err := fetcher.FetchPrices(context.Background(), params, nil)

	assert.NoError(t, err)
	assert.NotNil(t, tokenData)
	assert.Empty(t, tokenData)
}

func TestChunksFetcher_FetchPrices_DefaultValues(t *testing.T) {
	mockClient := new(MockAPIClient)
	mockClient.On("Healthy").Return(true)

	// Test with negative chunk size
	fetcher := NewChunksFetcher(mockClient, -1, 0)
	assert.Equal(t, DEFAULT_CHUNK_SIZE, fetcher.chunkSize)

	// Test with negative request delay
	fetcher = NewChunksFetcher(mockClient, 100, -1)
	assert.Equal(t, DEFAULT_REQUEST_DELAY*time.Millisecond, fetcher.requestDelay)
}

func TestChunksFetcher_FetchPrices_RequestDelay(t *testing.T) {
	mockClient := new(MockAPIClient)
	mockClient.On("Healthy").Return(true)

	// Set up expectations for two chunks
	mockClient.On("FetchPrices", cg.PriceParams{
		IDs:        []string{"token1", "token2"},
		Currencies: []string{"usd"},
	}).Return(map[string][]byte{
		"token1": []byte(`{"usd": 1.0}`),
		"token2": []byte(`{"usd": 2.0}`),
	}, nil)

	mockClient.On("FetchPrices", cg.PriceParams{
		IDs:        []string{"token3"},
		Currencies: []string{"usd"},
	}).Return(map[string][]byte{
		"token3": []byte(`{"usd": 3.0}`),
	}, nil)

	// Use a small delay for testing
	fetcher := NewChunksFetcher(mockClient, 2, 10)
	start := time.Now()
	params := cg.PriceParams{
		IDs:        []string{"token1", "token2", "token3"},
		Currencies: []string{"usd"},
	}
	tokenData, err := fetcher.FetchPrices(context.Background(), params, nil)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, tokenData)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}
