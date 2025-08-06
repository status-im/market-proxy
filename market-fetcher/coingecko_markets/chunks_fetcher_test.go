package coingecko_markets

import (
	"context"
	"errors"
	"testing"
	"time"

	api_mocks "github.com/status-im/market-proxy/coingecko_markets/mocks"
	"github.com/status-im/market-proxy/interfaces"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewChunksFetcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := api_mocks.NewMockAPIClient(ctrl)

	tests := []struct {
		name                 string
		chunkSize            int
		requestDelayMs       int
		expectedChunkSize    int
		expectedRequestDelay time.Duration
	}{
		{
			name:                 "Default values",
			chunkSize:            0,
			requestDelayMs:       -1,
			expectedChunkSize:    CHUNKS_DEFAULT_CHUNK_SIZE,
			expectedRequestDelay: CHUNKS_DEFAULT_REQUEST_DELAY * time.Millisecond,
		},
		{
			name:                 "Custom values",
			chunkSize:            100,
			requestDelayMs:       500,
			expectedChunkSize:    100,
			expectedRequestDelay: 500 * time.Millisecond,
		},
		{
			name:                 "Zero delay",
			chunkSize:            50,
			requestDelayMs:       0,
			expectedChunkSize:    50,
			expectedRequestDelay: 0,
		},
		{
			name:                 "Negative chunk size uses default",
			chunkSize:            -10,
			requestDelayMs:       100,
			expectedChunkSize:    CHUNKS_DEFAULT_CHUNK_SIZE,
			expectedRequestDelay: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewChunksFetcher(mockClient, tt.chunkSize, tt.requestDelayMs)

			assert.NotNil(t, fetcher)
			assert.Equal(t, mockClient, fetcher.apiClient)
			assert.Equal(t, tt.expectedChunkSize, fetcher.chunkSize)
			assert.Equal(t, tt.expectedRequestDelay, fetcher.requestDelay)
		})
	}
}

func TestChunksFetcher_FetchMarkets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := api_mocks.NewMockAPIClient(ctrl)

	// Sample market data responses
	sampleData1 := []byte(`{"id":"bitcoin","symbol":"btc","name":"Bitcoin","current_price":45000}`)
	sampleData2 := []byte(`{"id":"ethereum","symbol":"eth","name":"Ethereum","current_price":3000}`)
	sampleData3 := []byte(`{"id":"cardano","symbol":"ada","name":"Cardano","current_price":0.5}`)

	tests := []struct {
		name           string
		params         interfaces.MarketsParams
		chunkSize      int
		requestDelayMs int
		mockResponses  [][]byte
		mockErrors     []error
		expectedResult [][]byte
		expectedError  string
		expectedCalls  int
	}{
		{
			name: "Empty IDs list",
			params: interfaces.MarketsParams{
				IDs:      []string{},
				Currency: "usd",
			},
			chunkSize:      250,
			requestDelayMs: 0,
			expectedResult: [][]byte{},
			expectedCalls:  0,
		},
		{
			name: "Single chunk",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum"},
				Currency: "usd",
			},
			chunkSize:      250,
			requestDelayMs: 0,
			mockResponses: [][]byte{
				sampleData1, sampleData2,
			},
			mockErrors: []error{nil},
			expectedResult: [][]byte{
				sampleData1, sampleData2,
			},
			expectedCalls: 1,
		},
		{
			name: "Multiple chunks",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum", "cardano"},
				Currency: "usd",
			},
			chunkSize:      2,
			requestDelayMs: 0,
			mockResponses: [][]byte{
				sampleData1, sampleData2,
			},
			mockErrors: []error{nil, nil},
			expectedResult: [][]byte{
				sampleData1, sampleData2, sampleData3,
			},
			expectedCalls: 2,
		},
		{
			name: "API error in first chunk",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum"},
				Currency: "usd",
			},
			chunkSize:      250,
			requestDelayMs: 0,
			mockErrors:     []error{errors.New("API error")},
			expectedError:  "API error",
			expectedCalls:  1,
		},
		{
			name: "API error in second chunk",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum", "cardano"},
				Currency: "usd",
			},
			chunkSize:      2,
			requestDelayMs: 0,
			mockResponses: [][]byte{
				sampleData1, sampleData2,
			},
			mockErrors:    []error{nil, errors.New("API error in chunk 2")},
			expectedError: "API error in chunk 2",
			expectedCalls: 2,
		},
		{
			name: "With request delay",
			params: interfaces.MarketsParams{
				IDs:      []string{"bitcoin", "ethereum", "cardano"},
				Currency: "usd",
			},
			chunkSize:      2,
			requestDelayMs: 50, // Short delay for test
			mockResponses: [][]byte{
				sampleData1, sampleData2,
			},
			mockErrors: []error{nil, nil},
			expectedResult: [][]byte{
				sampleData1, sampleData2, sampleData3,
			},
			expectedCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewChunksFetcher(mockClient, tt.chunkSize, tt.requestDelayMs)

			// Setup mock expectations
			callCount := 0
			if tt.expectedCalls > 0 {
				mockClient.EXPECT().FetchPage(gomock.Any()).DoAndReturn(
					func(params interfaces.MarketsParams) ([][]byte, error) {
						if callCount < len(tt.mockErrors) && tt.mockErrors[callCount] != nil {
							err := tt.mockErrors[callCount]
							callCount++
							return nil, err
						}

						// Return appropriate response based on call count
						if callCount == 0 {
							// First chunk
							start := 0
							end := tt.chunkSize
							if end > len(tt.params.IDs) {
								end = len(tt.params.IDs)
							}
							callCount++
							if len(tt.mockResponses) >= end {
								return tt.mockResponses[start:end], nil
							}
							return tt.mockResponses, nil
						} else if callCount == 1 {
							// Second chunk
							callCount++
							return [][]byte{sampleData3}, nil
						}

						callCount++
						return [][]byte{}, nil
					},
				).Times(tt.expectedCalls)
			}

			// Execute test
			ctx := context.Background()
			startTime := time.Now()
			result, err := fetcher.FetchMarkets(ctx, tt.params, nil)
			duration := time.Since(startTime)

			// Verify results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expectedResult), len(result))

				// For multiple chunks with delay, verify timing
				if tt.expectedCalls > 1 && tt.requestDelayMs > 0 {
					expectedMinDuration := time.Duration(tt.requestDelayMs*(tt.expectedCalls-1)) * time.Millisecond
					assert.GreaterOrEqual(t, duration, expectedMinDuration)
				}
			}
		})
	}
}

func TestChunksFetcher_FetchMarkets_ParameterPassing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := api_mocks.NewMockAPIClient(ctrl)
	fetcher := NewChunksFetcher(mockClient, 2, 0)

	originalParams := interfaces.MarketsParams{
		IDs:                   []string{"bitcoin", "ethereum", "cardano"},
		Currency:              "eur",
		Order:                 "volume_desc",
		SparklineEnabled:      true,
		PriceChangePercentage: []string{"1h", "24h"},
		Category:              "defi",
	}

	// Mock to verify parameters are passed correctly
	mockClient.EXPECT().FetchPage(gomock.Any()).DoAndReturn(
		func(params interfaces.MarketsParams) ([][]byte, error) {
			// Verify that chunk parameters preserve original parameters
			assert.Equal(t, originalParams.Currency, params.Currency)
			assert.Equal(t, originalParams.Order, params.Order)
			assert.Equal(t, originalParams.SparklineEnabled, params.SparklineEnabled)
			assert.Equal(t, originalParams.PriceChangePercentage, params.PriceChangePercentage)
			assert.Equal(t, originalParams.Category, params.Category)

			// Verify chunk-specific parameters
			assert.Equal(t, 2, params.PerPage)
			assert.Equal(t, 1, params.Page)
			assert.LessOrEqual(t, len(params.IDs), 2) // Chunk size

			return [][]byte{
				[]byte(`{"id":"` + params.IDs[0] + `","symbol":"test","name":"Test"}`),
			}, nil
		},
	).Times(2) // 3 IDs with chunk size 2 = 2 chunks

	result, err := fetcher.FetchMarkets(context.Background(), originalParams, nil)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(result)) // 2 chunks = 2 results
}

func TestChunksFetcher_ChunkBoundaryCalculation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := api_mocks.NewMockAPIClient(ctrl)

	tests := []struct {
		name         string
		totalIDs     int
		chunkSize    int
		expectedCall int
	}{
		{
			name:         "Exact division",
			totalIDs:     6,
			chunkSize:    3,
			expectedCall: 2,
		},
		{
			name:         "With remainder",
			totalIDs:     7,
			chunkSize:    3,
			expectedCall: 3,
		},
		{
			name:         "Single item",
			totalIDs:     1,
			chunkSize:    5,
			expectedCall: 1,
		},
		{
			name:         "Large chunk size",
			totalIDs:     5,
			chunkSize:    100,
			expectedCall: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetcher := NewChunksFetcher(mockClient, tt.chunkSize, 0)

			// Generate IDs
			ids := make([]string, tt.totalIDs)
			for i := 0; i < tt.totalIDs; i++ {
				ids[i] = "token" + string(rune(i+'0'))
			}

			params := interfaces.MarketsParams{
				IDs:      ids,
				Currency: "usd",
			}

			// Mock expectations
			mockClient.EXPECT().FetchPage(gomock.Any()).Return(
				[][]byte{[]byte(`{"test":"data"}`)},
				nil,
			).Times(tt.expectedCall)

			_, err := fetcher.FetchMarkets(context.Background(), params, nil)
			assert.NoError(t, err)
		})
	}
}
