package coingecko_prices

import (
	"context"
	"log"
	"time"

	"github.com/status-im/market-proxy/coingecko_common"
	cg "github.com/status-im/market-proxy/interfaces"
)

const (
	// Default chunk size for token IDs
	DEFAULT_CHUNK_SIZE = 500
	// Default request delay in milliseconds
	DEFAULT_REQUEST_DELAY = 2000
)

// ChunksFetcher handles fetching prices in chunks
type ChunksFetcher struct {
	apiClient    APIClient
	chunkSize    int
	requestDelay time.Duration
}

// NewChunksFetcher creates a new chunks fetcher
func NewChunksFetcher(apiClient APIClient, chunkSize int, requestDelayMs int) *ChunksFetcher {
	var requestDelay time.Duration
	if requestDelayMs >= 0 {
		requestDelay = time.Duration(requestDelayMs) * time.Millisecond
	} else {
		// Negative delay means use default (2000ms)
		requestDelay = DEFAULT_REQUEST_DELAY * time.Millisecond
	}

	if chunkSize <= 0 {
		chunkSize = DEFAULT_CHUNK_SIZE
	}

	return &ChunksFetcher{
		apiClient:    apiClient,
		chunkSize:    chunkSize,
		requestDelay: requestDelay,
	}
}

// FetchPrices fetches prices for all token IDs in chunks
// onChunk callback is called for each successfully fetched chunk with the chunk data
func (f *ChunksFetcher) FetchPrices(ctx context.Context, params cg.PriceParams, onChunk func(map[string][]byte)) (map[string][]byte, error) {
	if len(params.IDs) == 0 {
		return make(map[string][]byte), nil
	}

	startTime := time.Now()
	numChunks := (len(params.IDs) + f.chunkSize - 1) / f.chunkSize
	log.Printf("IPricesService: Fetching prices for %d tokens in %d chunks", len(params.IDs), numChunks)

	// Create fetch function for chunks
	fetchFunc := func(ctx context.Context, chunk []string) (map[string][]byte, error) {
		chunkParams := cg.PriceParams{
			IDs:                  chunk,
			Currencies:           params.Currencies,
			IncludeMarketCap:     params.IncludeMarketCap,
			Include24hrVol:       params.Include24hrVol,
			Include24hrChange:    params.Include24hrChange,
			IncludeLastUpdatedAt: params.IncludeLastUpdatedAt,
			Precision:            params.Precision,
		}

		chunkData, err := f.apiClient.FetchPrices(chunkParams)
		if err != nil {
			log.Printf("IPricesService: Error fetching chunk: %v", err)
			return nil, err
		}

		// Call onChunk callback if provided
		if onChunk != nil {
			onChunk(chunkData)
		}

		return chunkData, nil
	}

	result, err := coingecko_common.ChunkMapFetcher(ctx, params.IDs, f.chunkSize, coingecko_common.MaxChunkStringLength, f.requestDelay, fetchFunc)
	if err != nil {
		return nil, err
	}

	tokensPerSecond := float64(len(params.IDs)) / time.Since(startTime).Seconds()
	log.Printf("IPricesService: Fetched prices for %d tokens in %d chunks (%.2f tokens/sec)",
		len(params.IDs), numChunks, tokensPerSecond)

	return result, nil
}
