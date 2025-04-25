package coingecko_leaderboard

import (
	"fmt"
	"log"
	"time"
)

const (
	// Default request delay in milliseconds
	DEFAULT_REQUEST_DELAY = 2000
)

// PaginatedFetcher handles fetching data with pagination support
type PaginatedFetcher struct {
	apiClient    APIClient
	maxLimit     int
	totalLimit   int
	requestDelay time.Duration // Delay between requests
}

// NewPaginatedFetcher creates a new paginated fetcher
func NewPaginatedFetcher(apiClient APIClient, totalLimit int, maxPerPage int, requestDelayMs int) *PaginatedFetcher {
	// Convert delay to time.Duration - allowing 0 as valid value
	var requestDelay time.Duration
	if requestDelayMs >= 0 {
		requestDelay = time.Duration(requestDelayMs) * time.Millisecond
	} else {
		// Negative delay means use default (2000ms)
		requestDelay = DEFAULT_REQUEST_DELAY * time.Millisecond
	}

	return &PaginatedFetcher{
		apiClient:    apiClient,
		maxLimit:     maxPerPage,
		totalLimit:   totalLimit,
		requestDelay: requestDelay,
	}
}

// FetchData fetches data with pagination
func (pf *PaginatedFetcher) FetchData() (*APIResponse, error) {
	params := pf.prepareFetchParams()

	// Track metrics
	startTime := time.Now()
	allItems := make([]CoinData, 0, pf.totalLimit)
	completedPages := 0

	// Fetch pages sequentially
	for page := 1; page <= params.totalPages; page++ {
		pageItems, shouldContinue, err := pf.processSinglePage(page, params, &allItems, &completedPages)

		// Handle any errors during page processing
		if err != nil {
			return pf.handlePageError(err, allItems)
		}

		// Add items from this page
		allItems = append(allItems, pageItems...)

		// Break if we shouldn't continue
		if !shouldContinue {
			break
		}

		// Handle delay between pages if needed
		pf.applyDelayIfNeeded(page, params.totalPages)
	}

	// Trim excess items if needed
	if len(allItems) > pf.totalLimit {
		allItems = allItems[:pf.totalLimit]
	}

	// Log summary
	pf.logSummary(startTime, allItems, completedPages)

	// Return results
	return &APIResponse{
		Data: allItems,
	}, nil
}

// fetchParams contains parameters needed for pagination
type fetchParams struct {
	totalPages int
	perPage    int
	totalLimit int
}

// prepareFetchParams calculates pagination parameters
func (pf *PaginatedFetcher) prepareFetchParams() *fetchParams {
	// Calculate total pages (will be 1 for small requests)
	totalPages := (pf.totalLimit + pf.maxLimit - 1) / pf.maxLimit // Ceiling division
	log.Printf("Fetcher: Fetching %d items in %d pages", pf.totalLimit, totalPages)

	return &fetchParams{
		totalPages: totalPages,
		perPage:    pf.maxLimit,
		totalLimit: pf.totalLimit,
	}
}

// processSinglePage processes a single page of data
// Returns: page items, should continue fetching flag, error
func (pf *PaginatedFetcher) processSinglePage(page int, params *fetchParams, allItems *[]CoinData, completedPages *int) ([]CoinData, bool, error) {
	// Calculate limit for this page
	pageLimit := pf.calculatePageLimit(page, params)

	// Log page fetch attempt
	log.Printf("Fetcher: Fetching page %d/%d with limit %d", page, params.totalPages, pageLimit)
	pageStartTime := time.Now()

	// Fetch the page
	pageResponse, err := pf.fetchSinglePage(page, pageLimit)
	if err != nil {
		return nil, false, err
	}

	// Process successful response
	pageTime := time.Since(pageStartTime)

	// No items in response
	if pageResponse == nil || len(pageResponse.Data) == 0 {
		log.Printf("Fetcher: Got empty page %d, stopping pagination", page)
		return []CoinData{}, false, nil
	}

	// Track successful page
	(*completedPages)++

	log.Printf("Fetcher: Completed page %d/%d with %d items in %.2fs",
		page, params.totalPages, len(pageResponse.Data), pageTime.Seconds())

	// Check if we've reached our limit
	if len(*allItems)+len(pageResponse.Data) >= pf.totalLimit {
		log.Printf("Fetcher: Reached target limit of %d items", pf.totalLimit)
		return pageResponse.Data, false, nil
	}

	return pageResponse.Data, true, nil
}

// calculatePageLimit calculates the limit for a specific page
func (pf *PaginatedFetcher) calculatePageLimit(page int, params *fetchParams) int {
	if page == params.totalPages {
		// Last page might need fewer items
		return params.totalLimit - (page-1)*params.perPage
	}
	return params.perPage
}

// handlePageError handles errors during page processing
func (pf *PaginatedFetcher) handlePageError(err error, allItems []CoinData) (*APIResponse, error) {
	log.Printf("Fetcher: Error fetching page: %v", err)

	// If we have some data already, return what we have
	if len(allItems) > 0 {
		log.Printf("Fetcher: Returning partial data (%d items)", len(allItems))
		return &APIResponse{Data: allItems}, nil
	}

	// If no data at all, return the error
	return nil, fmt.Errorf("failed to fetch data: %v", err)
}

// applyDelayIfNeeded applies delay between page requests if configured
func (pf *PaginatedFetcher) applyDelayIfNeeded(currentPage, totalPages int) {
	// If there are more pages to fetch, wait before the next request
	// Only wait if requestDelay > 0
	if currentPage < totalPages && pf.requestDelay > 0 {
		log.Printf("Fetcher: Waiting for %.2fs before fetching next page", pf.requestDelay.Seconds())
		time.Sleep(pf.requestDelay)
	} else if currentPage < totalPages {
		log.Printf("Fetcher: No delay configured, fetching next page immediately")
	}
}

// logSummary logs a summary of the fetch operation
func (pf *PaginatedFetcher) logSummary(startTime time.Time, items []CoinData, completedPages int) {
	totalTime := time.Since(startTime)
	itemsPerSecond := float64(len(items)) / totalTime.Seconds()
	log.Printf("Fetcher: Fetched %d/%d items in %d pages (%.2f items/sec)",
		len(items), pf.totalLimit, completedPages, itemsPerSecond)
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
