package coingecko_prices

import (
	"fmt"
	"log"
	"time"
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
func (f *ChunksFetcher) FetchPrices(params PriceParams) (map[string]map[string]float64, error) {
	// Track overall execution time
	startTime := time.Now()

	// Handle empty input
	if len(params.IDs) == 0 {
		return make(map[string]map[string]float64), nil
	}

	// Calculate number of chunks
	numChunks := (len(params.IDs) + f.chunkSize - 1) / f.chunkSize
	log.Printf("Fetcher: Fetching prices for %d tokens in %d chunks", len(params.IDs), numChunks)

	// Initialize result map
	result := make(map[string]map[string]float64)

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
		log.Printf("Fetcher: Fetching chunk %d/%d with %d tokens", i+1, numChunks, len(chunk))

		// Fetch prices for this chunk
		chunkStartTime := time.Now()
		chunkParams := PriceParams{
			IDs:        chunk,
			Currencies: params.Currencies,
			// Use the same optional parameters as the original request
			IncludeMarketCap:     params.IncludeMarketCap,
			Include24hrVol:       params.Include24hrVol,
			Include24hrChange:    params.Include24hrChange,
			IncludeLastUpdatedAt: params.IncludeLastUpdatedAt,
			Precision:            params.Precision,
		}
		prices, err := f.apiClient.FetchPrices(chunkParams)
		if err != nil {
			log.Printf("Fetcher: Error fetching chunk: %v", err)
			return nil, err
		}
		duration := time.Since(chunkStartTime)
		log.Printf("Fetcher: Completed chunk %d/%d with %d tokens in %.2fs", i+1, numChunks, len(chunk), duration.Seconds())

		// Merge prices into result
		for currency, currencyPrices := range prices {
			if _, exists := result[currency]; !exists {
				result[currency] = make(map[string]float64)
			}
			for tokenId, price := range currencyPrices {
				result[currency][tokenId] = price
			}
		}

		// Add delay between chunks if not the last chunk
		if i < numChunks-1 && f.requestDelay > 0 {
			log.Printf("Fetcher: Waiting for %v before fetching next chunk", f.requestDelay)
			time.Sleep(f.requestDelay)
		} else if i < numChunks-1 {
			log.Printf("Fetcher: No delay configured, fetching next chunk immediately")
		}
	}

	// Log completion
	tokensPerSecond := float64(len(params.IDs)) / time.Since(startTime).Seconds()
	log.Printf("Fetcher: Fetched prices for %d tokens in %d chunks (%.2f tokens/sec)",
		len(params.IDs), numChunks, tokensPerSecond)

	return result, nil
}

// fetchChunk fetches prices for a single chunk of token IDs
func (cf *ChunksFetcher) fetchChunk(params PriceParams) (map[string]map[string]float64, error) {
	return cf.apiClient.FetchPrices(params)
}

// handleChunkError handles errors during chunk processing
func (cf *ChunksFetcher) handleChunkError(err error, allPrices map[string]map[string]float64) (map[string]map[string]float64, error) {
	log.Printf("Fetcher: Error fetching chunk: %v", err)

	// If we have some data already, return what we have
	if len(allPrices) > 0 {
		log.Printf("Fetcher: Returning partial data (%d tokens)", len(allPrices))
		return allPrices, nil
	}

	// If no data at all, return the error
	return nil, fmt.Errorf("failed to fetch prices: %v", err)
}

// applyDelayIfNeeded applies delay between chunk requests if configured
func (cf *ChunksFetcher) applyDelayIfNeeded(currentChunk, totalChunks int) {
	// If there are more chunks to fetch, wait before the next request
	// Only wait if requestDelay > 0
	if currentChunk < totalChunks && cf.requestDelay > 0 {
		log.Printf("Fetcher: Waiting for %.2fs before fetching next chunk", cf.requestDelay.Seconds())
		time.Sleep(cf.requestDelay)
	} else if currentChunk < totalChunks {
		log.Printf("Fetcher: No delay configured, fetching next chunk immediately")
	}
}

// logSummary logs a summary of the fetch operation
func (cf *ChunksFetcher) logSummary(startTime time.Time, prices map[string]map[string]float64, completedChunks int) {
	totalTime := time.Since(startTime)
	tokensPerSecond := float64(len(prices)) / totalTime.Seconds()
	log.Printf("Fetcher: Fetched prices for %d tokens in %d chunks (%.2f tokens/sec)",
		len(prices), completedChunks, tokensPerSecond)
}
