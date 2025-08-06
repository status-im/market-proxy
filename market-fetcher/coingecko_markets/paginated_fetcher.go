package coingecko_markets

import (
	"fmt"
	"log"
	"time"

	"github.com/status-im/market-proxy/interfaces"
)

// PageData represents data for a single page - moved here since it's used by PaginatedFetcher
type PageData struct {
	Page int
	Data [][]byte
}

const (
	// Default request delay in milliseconds
	DEFAULT_REQUEST_DELAY = 2000
	DEFAULT_PER_PAGE      = 250
)

// PaginatedFetcher handles fetching data with pagination support
type PaginatedFetcher struct {
	apiClient    APIClient
	pageFrom     int
	pageTo       int
	perPage      int
	requestDelay time.Duration            // Delay between requests
	params       interfaces.MarketsParams // Markets parameters
}

// NewPaginatedFetcher creates a new paginated fetcher
func NewPaginatedFetcher(apiClient APIClient, pageFrom int, pageTo int, requestDelayMs int, params interfaces.MarketsParams) *PaginatedFetcher {
	// Convert delay to time.Duration - allowing 0 as valid value
	var requestDelay time.Duration
	if requestDelayMs >= 0 {
		requestDelay = time.Duration(requestDelayMs) * time.Millisecond
	} else {
		// Negative delay means use default (2000ms)
		requestDelay = DEFAULT_REQUEST_DELAY * time.Millisecond
	}

	// Set default parameters if not provided
	if params.Currency == "" {
		params.Currency = "usd"
	}
	if params.Order == "" {
		params.Order = "market_cap_desc"
	}
	if params.PerPage <= 0 {
		params.PerPage = DEFAULT_PER_PAGE
	}

	return &PaginatedFetcher{
		apiClient:    apiClient,
		pageFrom:     pageFrom,
		pageTo:       pageTo,
		perPage:      params.PerPage,
		requestDelay: requestDelay,
		params:       params,
	}
}

// FetchPages fetches data with pagination and returns page-data pairs
// onPage callback is called for each successfully fetched page with the page data
func (pf *PaginatedFetcher) FetchPages(onPage func(PageData)) ([]PageData, error) {
	params := pf.prepareFetchParams()

	// Track metrics
	startTime := time.Now()
	allPages := make([]PageData, 0, params.pageTo-params.pageFrom+1)
	completedPages := 0

	// Fetch pages sequentially from pageFrom to pageTo
	for page := pf.pageFrom; page <= pf.pageTo; page++ {
		pageItems, shouldContinue, err := pf.processSinglePage(page, params, 0, &completedPages)
		if err != nil {
			return pf.handlePagesError(err, allPages)
		}

		if len(pageItems) > 0 {
			pageData := PageData{
				Page: page,
				Data: pageItems,
			}
			allPages = append(allPages, pageData)

			if onPage != nil {
				onPage(pageData)
			}
		}

		if !shouldContinue {
			break
		}

		pf.applyDelayIfNeeded(page, pf.pageTo)
	}

	pf.logPagesSummary(startTime, allPages, completedPages)
	return allPages, nil
}

// FetchData fetches data with pagination
func (pf *PaginatedFetcher) FetchData() ([][]byte, error) {
	// Use FetchPages and flatten the results
	pagesData, err := pf.FetchPages(nil)
	if err != nil {
		return nil, err
	}

	// Flatten pages data into a single slice
	var allItems [][]byte
	for _, pageData := range pagesData {
		allItems = append(allItems, pageData.Data...)
	}

	return allItems, nil
}

// fetchParams contains parameters needed for pagination
type fetchParams struct {
	pageFrom       int
	pageTo         int
	perPage        int
	estimatedItems int
}

// prepareFetchParams calculates pagination parameters
func (pf *PaginatedFetcher) prepareFetchParams() *fetchParams {
	// Calculate estimated items based on page range
	totalPages := pf.pageTo - pf.pageFrom + 1
	estimatedItems := totalPages * pf.perPage
	log.Printf("MarketsFetcher: Fetching pages %d-%d (estimated %d items)", pf.pageFrom, pf.pageTo, estimatedItems)

	return &fetchParams{
		pageFrom:       pf.pageFrom,
		pageTo:         pf.pageTo,
		perPage:        pf.perPage,
		estimatedItems: estimatedItems,
	}
}

// processSinglePage processes a single page of data
// Returns: page items, should continue fetching flag, error
func (pf *PaginatedFetcher) processSinglePage(page int, params *fetchParams, currentItemsCount int, completedPages *int) ([][]byte, bool, error) {
	pageLimit := pf.perPage
	totalPages := params.pageTo - params.pageFrom + 1
	log.Printf("MarketsFetcher: Fetching page %d (page %d/%d) with limit %d", page, page-params.pageFrom+1, totalPages, pageLimit)
	pageStartTime := time.Now()

	// Fetch the page
	pageResponse, err := pf.fetchSinglePage(page, pageLimit)
	if err != nil {
		return nil, false, err
	}

	pageTime := time.Since(pageStartTime)

	if len(pageResponse) == 0 {
		log.Printf("MarketsFetcher: Got empty page %d, stopping pagination", page)
		return [][]byte{}, false, nil
	}
	(*completedPages)++

	log.Printf("MarketsFetcher: Completed page %d with %d items in %.2fs",
		page, len(pageResponse), pageTime.Seconds())

	return pageResponse, true, nil
}

// handlePageError handles errors during page processing
func (pf *PaginatedFetcher) handlePageError(err error, allItems [][]byte) ([][]byte, error) {
	log.Printf("MarketsFetcher: Error fetching page: %v", err)

	// If we have some data already, return what we have
	if len(allItems) > 0 {
		log.Printf("MarketsFetcher: Returning partial data (%d items)", len(allItems))
		return allItems, nil
	}

	// If no data at all, return the error
	return nil, fmt.Errorf("failed to fetch data: %v", err)
}

// handlePagesError handles errors during pages processing
func (pf *PaginatedFetcher) handlePagesError(err error, allPages []PageData) ([]PageData, error) {
	log.Printf("MarketsFetcher: Error fetching page: %v", err)

	// If we have some data already, return what we have
	if len(allPages) > 0 {
		totalItems := 0
		for _, page := range allPages {
			totalItems += len(page.Data)
		}
		log.Printf("MarketsFetcher: Returning partial data (%d pages, %d items)", len(allPages), totalItems)
		return allPages, nil
	}

	// If no data at all, return the error
	return nil, fmt.Errorf("failed to fetch data: %v", err)
}

// logPagesSummary logs a summary of the pages fetch operation
func (pf *PaginatedFetcher) logPagesSummary(startTime time.Time, pages []PageData, completedPages int) {
	totalTime := time.Since(startTime)
	totalItems := 0
	for _, page := range pages {
		totalItems += len(page.Data)
	}
	itemsPerSecond := float64(totalItems) / totalTime.Seconds()
	log.Printf("MarketsFetcher: Fetched %d items from pages %d-%d in %d pages (%.2f items/sec)",
		totalItems, pf.pageFrom, pf.pageTo, completedPages, itemsPerSecond)
}

// applyDelayIfNeeded applies delay between page requests if configured
func (pf *PaginatedFetcher) applyDelayIfNeeded(currentPage, totalPages int) {
	// If there are more pages to fetch, wait before the next request
	// Only wait if requestDelay > 0
	if currentPage < totalPages && pf.requestDelay > 0 {
		log.Printf("MarketsFetcher: Waiting for %.2fs before fetching next page", pf.requestDelay.Seconds())
		time.Sleep(pf.requestDelay)
	} else if currentPage < totalPages {
		log.Printf("MarketsFetcher: No delay configured, fetching next page immediately")
	}
}

// logSummary logs a summary of the fetch operation
func (pf *PaginatedFetcher) logSummary(startTime time.Time, items [][]byte, completedPages int) {
	totalTime := time.Since(startTime)
	itemsPerSecond := float64(len(items)) / totalTime.Seconds()
	log.Printf("MarketsFetcher: Fetched %d items from pages %d-%d in %d pages (%.2f items/sec)",
		len(items), pf.pageFrom, pf.pageTo, completedPages, itemsPerSecond)
}

// fetchSinglePage fetches a single page of data using the API client
func (pf *PaginatedFetcher) fetchSinglePage(page, limit int) ([][]byte, error) {
	// Create a copy of params and set page and limit
	params := pf.params
	params.Page = page
	params.PerPage = limit

	// Fetch raw bytes from API
	rawItems, err := pf.apiClient.FetchPage(params)
	if err != nil {
		return nil, err
	}

	return rawItems, nil
}
