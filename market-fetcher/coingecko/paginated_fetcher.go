package coingecko

import (
	"fmt"
	"log"
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
	// Calculate how many pages we need
	totalLimit := pf.totalLimit
	perPage := pf.maxLimit

	// Calculate total pages (will be 1 for small requests)
	totalPages := (totalLimit + perPage - 1) / perPage // Ceiling division
	log.Printf("Fetcher: Fetching %d items in %d pages", totalLimit, totalPages)

	// Track metrics
	startTime := time.Now()
	allItems := make([]CoinData, 0, totalLimit)
	completedPages := 0

	// Fetch pages sequentially
	for page := 1; page <= totalPages; page++ {
		// Calculate limit for this page
		pageLimit := perPage
		if page == totalPages {
			// Last page might need fewer items
			pageLimit = totalLimit - (page-1)*perPage
		}

		// Log page fetch attempt
		log.Printf("Fetcher: Fetching page %d/%d with limit %d", page, totalPages, pageLimit)
		pageStartTime := time.Now()

		// Fetch the page
		pageResponse, err := pf.fetchSinglePage(page, pageLimit)

		// Handle errors
		if err != nil {
			log.Printf("Fetcher: Error fetching page %d: %v", page, err)

			// If we have some data already, return what we have
			if len(allItems) > 0 {
				log.Printf("Fetcher: Returning partial data (%d items)", len(allItems))
				return &APIResponse{Data: allItems}, nil
			}

			// If no data at all, return the error
			return nil, fmt.Errorf("failed to fetch data: %v", err)
		}

		// Process successful response
		pageTime := time.Since(pageStartTime)

		// No items in response
		if pageResponse == nil || len(pageResponse.Data) == 0 {
			log.Printf("Fetcher: Got empty page %d, stopping pagination", page)
			break
		}

		// Add items to our collection
		allItems = append(allItems, pageResponse.Data...)
		completedPages++

		log.Printf("Fetcher: Completed page %d/%d with %d items in %.2fs",
			page, totalPages, len(pageResponse.Data), pageTime.Seconds())

		// Check if we've reached our limit
		if len(allItems) >= totalLimit {
			log.Printf("Fetcher: Reached target limit of %d items", totalLimit)
			break
		}
	}

	// Trim excess items if needed
	if len(allItems) > totalLimit {
		allItems = allItems[:totalLimit]
	}

	// Log summary
	totalTime := time.Since(startTime)
	itemsPerSecond := float64(len(allItems)) / totalTime.Seconds()
	log.Printf("Fetcher: Fetched %d/%d items in %d pages (%.2f items/sec)",
		len(allItems), totalLimit, completedPages, itemsPerSecond)

	// Return results
	return &APIResponse{
		Data: allItems,
	}, nil
}

// fetchSinglePage fetches a single page of data using the API client
func (pf *PaginatedFetcher) fetchSinglePage(page, limit int) (*APIResponse, error) {
	items, err := pf.apiClient.FetchPage(page, limit)
	if err != nil {
		return nil, err
	}

	return &APIResponse{
		Data: items,
	}, nil
}
