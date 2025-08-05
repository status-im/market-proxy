package coingecko_prices

import (
	"log"
	"time"

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
	// Convert delay to time.Duration - allowing 0 as valid value
	var requestDelay time.Duration
	if requestDelayMs >= 0 {
		requestDelay = time.Duration(requestDelayMs) * time.Millisecond
	} else {
		// Negative delay means use default (2000ms)
		requestDelay = DEFAULT_REQUEST_DELAY * time.Millisecond
	}

	// Use default chunk size if not specified
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
func (f *ChunksFetcher) FetchPrices(params cg.PriceParams) (map[string][]byte, error) {
	// Track overall execution time
	startTime := time.Now()

	// Handle empty input
	if len(params.IDs) == 0 {
		return make(map[string][]byte), nil
	}

	// Calculate number of chunks
	numChunks := (len(params.IDs) + f.chunkSize - 1) / f.chunkSize
	log.Printf("CoingeckoPricesService: Fetching prices for %d tokens in %d chunks", len(params.IDs), numChunks)

	// Initialize result map
	result := make(map[string][]byte)

	// Process each chunk
	for i := 0; i < numChunks; i++ {
		// Calculate chunk boundaries
		start := i * f.chunkSize
		end := start + f.chunkSize
		if end > len(params.IDs) {
			end = len(params.IDs)
		}

		// Get chunk of token IDs
		chunk := params.IDs[start:end]
		log.Printf("CoingeckoPricesService: Fetching chunk %d/%d with %d tokens", i+1, numChunks, len(chunk))

		// Fetch prices for this chunk
		chunkStartTime := time.Now()
		chunkParams := cg.PriceParams{
			IDs:        chunk,
			Currencies: params.Currencies,
			// Use the same optional parameters as the original request
			IncludeMarketCap:     params.IncludeMarketCap,
			Include24hrVol:       params.Include24hrVol,
			Include24hrChange:    params.Include24hrChange,
			IncludeLastUpdatedAt: params.IncludeLastUpdatedAt,
			Precision:            params.Precision,
		}
		chunkData, err := f.apiClient.FetchPrices(chunkParams)
		if err != nil {
			log.Printf("CoingeckoPricesService: Error fetching chunk: %v", err)
			return nil, err
		}
		duration := time.Since(chunkStartTime)
		log.Printf("CoingeckoPricesService: Completed chunk %d/%d with %d tokens in %.2fs", i+1, numChunks, len(chunk), duration.Seconds())

		// Merge chunk data into result
		for tokenId, tokenData := range chunkData {
			result[tokenId] = tokenData
		}

		// Add delay between chunks if not the last chunk
		if i < numChunks-1 && f.requestDelay > 0 {
			log.Printf("CoingeckoPricesService: Waiting for %v before fetching next chunk", f.requestDelay)
			time.Sleep(f.requestDelay)
		} else if i < numChunks-1 {
			log.Printf("CoingeckoPricesService: No delay configured, fetching next chunk immediately")
		}
	}

	// Log completion
	tokensPerSecond := float64(len(params.IDs)) / time.Since(startTime).Seconds()
	log.Printf("CoingeckoPricesService: Fetched prices for %d tokens in %d chunks (%.2f tokens/sec)",
		len(params.IDs), numChunks, tokensPerSecond)

	return result, nil
}
