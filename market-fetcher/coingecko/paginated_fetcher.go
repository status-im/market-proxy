package coingecko

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// PaginatedFetcher handles fetching data with pagination support
type PaginatedFetcher struct {
	apiClient  APIClient
	maxLimit   int
	totalLimit int
}

// NewPaginatedFetcher creates a new paginated fetcher
func NewPaginatedFetcher(apiClient APIClient, totalLimit int, maxPerPage int) *PaginatedFetcher {
	return &PaginatedFetcher{
		apiClient:  apiClient,
		maxLimit:   maxPerPage,
		totalLimit: totalLimit,
	}
}

// FetchData fetches data with pagination
func (pf *PaginatedFetcher) FetchData() (*APIResponse, error) {
	// Create a random number generator with current time seed
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Calculate how many pages we need
	totalLimit := pf.totalLimit
	perPage := pf.maxLimit

	if totalLimit <= perPage {
		// If limit is less than max per page, just fetch one page
		log.Printf("Fetcher: Small request, fetching single page with %d items", totalLimit)
		return pf.fetchSinglePage(1, totalLimit)
	}

	// Need multiple pages
	totalPages := (totalLimit + perPage - 1) / perPage // Ceiling division
	log.Printf("Fetcher: Fetching %d items in %d pages using sequential requests", totalLimit, totalPages)

	// Use a rate limiter to control API request speed
	// Start with 1 request per 1.5 seconds (conservative)
	requestInterval := 1500 * time.Millisecond

	// Track metrics
	startTime := time.Now()
	completedPages := 0
	retriedPages := 0
	allItems := make([]interface{}, 0, totalLimit)

	// Fetch pages sequentially with rate limiting
	for page := 1; page <= totalPages; page++ {
		// Calculate limit for this page
		pageLimit := perPage
		if page == totalPages {
			pageLimit = totalLimit - (page-1)*perPage
		}

		if page > 1 {
			// Add jitter to the delay
			jitter := time.Duration(r.Intn(500)) * time.Millisecond // 0-500ms jitter
			delay := requestInterval + jitter
			log.Printf("Fetcher: Rate limiting - waiting %.2fs before fetching page %d", delay.Seconds(), page)
			time.Sleep(delay)
		}

		pageStartTime := time.Now()
		log.Printf("Fetcher: Starting fetch for page %d with limit %d", page, pageLimit)

		// Fetch page from API client
		pageItems, err := pf.fetchSinglePage(page, pageLimit)
		if err != nil {
			// If this was a rate limit error and we've only completed a few pages,
			// increase the interval for subsequent requests
			if err.Error() != "" && completedPages < 3 {
				oldInterval := requestInterval
				// Double the interval
				requestInterval = requestInterval * 2
				log.Printf("Fetcher: Rate limit hit early, increasing request interval from %.1fs to %.1fs",
					oldInterval.Seconds(), requestInterval.Seconds())

				// If we hit rate limits very early, it might be better to fetch fewer pages
				if completedPages < 2 && totalPages > 5 {
					newTotal := totalPages / 2
					if newTotal < 3 {
						newTotal = 3
					}
					log.Printf("Fetcher: Rate limit hit very early, reducing target from %d to %d pages",
						totalPages, newTotal)
					totalPages = newTotal
					totalLimit = newTotal * perPage
				}

				// Wait longer before retrying
				extraWait := 5 * time.Second
				log.Printf("Fetcher: Waiting an extra %.1fs before continuing", extraWait.Seconds())
				time.Sleep(extraWait)

				// Retry this page after adapting
				page--
				retriedPages++
				continue
			}

			// For other errors or if we've tried to adapt multiple times, log and continue
			if retriedPages > 5 {
				log.Printf("Fetcher: Too many retries, continuing with partial data")
			} else {
				log.Printf("Fetcher: Error fetching page %d: %v", page, err)
			}

			// If we have some data already, continue with what we have
			if len(allItems) > 0 {
				log.Printf("Fetcher: Continuing with partial data (%d items so far)", len(allItems))
				break
			}

			// If we have no data at all, propagate the error
			return nil, fmt.Errorf("failed to fetch initial data: %v", err)
		}

		if pageItems != nil && len(pageItems.Data) > 0 {
			// Add items from this page to the result
			for _, coin := range pageItems.Data {
				allItems = append(allItems, coin)
			}

			// Update metrics
			completedPages++
			progress := float64(completedPages) / float64(totalPages) * 100
			pageTime := time.Since(pageStartTime)
			log.Printf("Fetcher: Completed page %d/%d (%.1f%%) with %d items in %.2fs",
				page, totalPages, progress, len(pageItems.Data), pageTime.Seconds())

			// If we've got enough data, stop
			if len(allItems) >= totalLimit {
				log.Printf("Fetcher: Reached target limit of %d items after %d pages", totalLimit, completedPages)
				break
			}
		} else {
			log.Printf("Fetcher: Warning - received empty page %d", page)
		}
	}

	// Trim to requested limit if we got more
	if len(allItems) > totalLimit {
		allItems = allItems[:totalLimit]
	}

	totalTime := time.Since(startTime)
	log.Printf("Fetcher: Successfully fetched %d items in %.2fs (%.2f items/sec)",
		len(allItems), totalTime.Seconds(), float64(len(allItems))/totalTime.Seconds())

	// If we didn't get all the data we wanted, but have some, return what we have with a warning
	if len(allItems) < totalLimit && len(allItems) > 0 {
		log.Printf("Fetcher: WARNING - Only fetched %d/%d requested items due to rate limits",
			len(allItems), totalLimit)
	}

	// Convert generic items back to CoinData
	coinItems := make([]CoinData, len(allItems))
	for i, item := range allItems {
		if coin, ok := item.(CoinData); ok {
			coinItems[i] = coin
		}
	}

	// Create final response
	return &APIResponse{
		Data: coinItems,
	}, nil
}

// fetchSinglePage fetches a single page of data using the API client
func (pf *PaginatedFetcher) fetchSinglePage(page, limit int) (*APIResponse, error) {
	// Get data from API client
	items, err := pf.apiClient.FetchPage(page, limit)
	if err != nil {
		return nil, err
	}

	// Convert to CoinData slice
	coinItems := make([]CoinData, 0, len(items))
	for _, item := range items {
		if coin, ok := item.(CoinData); ok {
			coinItems = append(coinItems, coin)
		}
	}

	return &APIResponse{
		Data: coinItems,
	}, nil
}
