package coingecko_markets

import (
	"context"
	"log"
	"time"

	"github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/interfaces"
)

const (
	CHUNKS_DEFAULT_CHUNK_SIZE    = 250
	CHUNKS_DEFAULT_REQUEST_DELAY = 1000 // ms
)

// ChunksFetcher handles fetching markets data in chunks
type ChunksFetcher struct {
	apiClient    IAPIClient
	chunkSize    int
	requestDelay time.Duration
}

func NewChunksFetcher(apiClient IAPIClient, chunkSize int, requestDelayMs int) *ChunksFetcher {
	var requestDelay time.Duration
	if requestDelayMs >= 0 {
		requestDelay = time.Duration(requestDelayMs) * time.Millisecond
	} else {
		// Negative delay means use default (1000ms)
		requestDelay = CHUNKS_DEFAULT_REQUEST_DELAY * time.Millisecond
	}

	if chunkSize <= 0 {
		chunkSize = CHUNKS_DEFAULT_CHUNK_SIZE
	}

	return &ChunksFetcher{
		apiClient:    apiClient,
		chunkSize:    chunkSize,
		requestDelay: requestDelay,
	}
}

// FetchMarkets fetches markets data for all token IDs in chunks
// onChunk callback is called for each successfully fetched chunk with the chunk data
func (f *ChunksFetcher) FetchMarkets(ctx context.Context, params interfaces.MarketsParams, onChunk func([][]byte)) ([][]byte, error) {
	if len(params.IDs) == 0 {
		return [][]byte{}, nil
	}

	startTime := time.Now()
	numChunks := (len(params.IDs) + f.chunkSize - 1) / f.chunkSize
	log.Printf("CoingeckoMarketsChunksFetcher: Fetching markets data for %d tokens in %d chunks", len(params.IDs), numChunks)

	fetchFunc := func(ctx context.Context, chunk []string) ([][]byte, error) {
		log.Printf("CoingeckoMarketsChunksFetcher: Fetching chunk with %d tokens", len(chunk))
		chunkStartTime := time.Now()

		chunkParams := interfaces.MarketsParams{
			IDs:      chunk,
			Currency: params.Currency,
			// Use the same optional parameters as the original request
			Order:                 params.Order,
			PerPage:               f.chunkSize, // Use chunk size as per_page
			Page:                  1,           // Always use page 1 for chunk requests
			SparklineEnabled:      params.SparklineEnabled,
			PriceChangePercentage: params.PriceChangePercentage,
			Category:              params.Category,
		}

		chunkData, err := f.apiClient.FetchPage(chunkParams)
		if err != nil {
			log.Printf("CoingeckoMarketsChunksFetcher: Error fetching chunk: %v", err)
			return nil, err
		}

		duration := time.Since(chunkStartTime)
		log.Printf("CoingeckoMarketsChunksFetcher: Completed chunk with %d tokens in %.2fs", len(chunk), duration.Seconds())

		if onChunk != nil {
			onChunk(chunkData)
		}

		return chunkData, nil
	}

	result, err := coingecko_common.ChunkArrayFetcher(ctx, params.IDs, f.chunkSize, f.requestDelay, fetchFunc)
	if err != nil {
		return nil, err
	}

	tokensPerSecond := float64(len(params.IDs)) / time.Since(startTime).Seconds()
	log.Printf("CoingeckoMarketsChunksFetcher: Fetched markets data for %d tokens in %d chunks (%.2f tokens/sec)",
		len(params.IDs), numChunks, tokensPerSecond)

	return result, nil
}
