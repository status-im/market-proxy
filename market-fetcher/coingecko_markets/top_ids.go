package coingecko_markets

import (
	"log"
	"sync"
)

// TopIdsManager manages top token IDs with efficient page-based updates
type TopIdsManager struct {
	mu sync.RWMutex

	// Map from page number to token IDs for that page
	pageToIds map[int][]string

	// Cached vector of all top IDs in order (built from pages)
	topIds []string

	// Track if topIds needs to be rebuilt
	dirty bool
}

// NewTopIdsManager creates a new TopIdsManager
func NewTopIdsManager() *TopIdsManager {
	return &TopIdsManager{
		pageToIds: make(map[int][]string),
		topIds:    make([]string, 0),
		dirty:     false,
	}
}

// UpdatePageIds updates token IDs for a specific page
// This method efficiently updates the internal topIds vector
func (t *TopIdsManager) UpdatePageIds(page int, tokenIds []string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Store page IDs
	t.pageToIds[page] = make([]string, len(tokenIds))
	copy(t.pageToIds[page], tokenIds)

	// Mark as dirty for rebuild
	t.dirty = true

	log.Printf("Updated page %d with %d token IDs", page, len(tokenIds))
}

// UpdatePagesFromPageData updates multiple pages from PageData slice
func (t *TopIdsManager) UpdatePagesFromPageData(pagesData []PageData) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, pageData := range pagesData {
		// Extract token IDs from page data
		tokenIds := extractTokenIDsFromPageData(pageData.Data)

		// Store page IDs
		t.pageToIds[pageData.Page] = make([]string, len(tokenIds))
		copy(t.pageToIds[pageData.Page], tokenIds)

		log.Printf("Updated page %d with %d token IDs from PageData", pageData.Page, len(tokenIds))
	}

	// Mark as dirty for rebuild
	t.dirty = true
}

// GetTopIds returns the current top IDs vector up to the specified limit
// Rebuilds the vector if it's dirty
func (t *TopIdsManager) GetTopIds(limit int) []string {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Rebuild if dirty
	if t.dirty {
		t.rebuildTopIds()
		t.dirty = false
	}

	// Return limited results
	if limit <= 0 || limit > len(t.topIds) {
		result := make([]string, len(t.topIds))
		copy(result, t.topIds)
		return result
	}

	result := make([]string, limit)
	copy(result, t.topIds[:limit])
	return result
}

// GetPageIds returns token IDs for a specific page
func (t *TopIdsManager) GetPageIds(page int) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ids, exists := t.pageToIds[page]
	if !exists {
		return []string{}
	}

	result := make([]string, len(ids))
	copy(result, ids)
	return result
}

// GetAvailablePages returns a slice of page numbers that have data
func (t *TopIdsManager) GetAvailablePages() []int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	pages := make([]int, 0, len(t.pageToIds))
	for page := range t.pageToIds {
		pages = append(pages, page)
	}

	return pages
}

// GetTotalTokenCount returns the total number of tokens across all pages
func (t *TopIdsManager) GetTotalTokenCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.dirty {
		// Count from pages
		total := 0
		for _, ids := range t.pageToIds {
			total += len(ids)
		}
		return total
	}

	return len(t.topIds)
}

// rebuildTopIds rebuilds the topIds vector from page data
// Must be called with lock held
func (t *TopIdsManager) rebuildTopIds() {
	if len(t.pageToIds) == 0 {
		t.topIds = make([]string, 0)
		return
	}

	// Find the highest page number to determine order
	maxPage := 0
	for page := range t.pageToIds {
		if page > maxPage {
			maxPage = page
		}
	}

	// Rebuild topIds in page order
	var newTopIds []string
	for page := 1; page <= maxPage; page++ {
		if ids, exists := t.pageToIds[page]; exists {
			newTopIds = append(newTopIds, ids...)
		}
	}

	t.topIds = newTopIds
	log.Printf("Rebuilt topIds vector with %d tokens from %d pages", len(t.topIds), len(t.pageToIds))
}

// Clear removes all data
func (t *TopIdsManager) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pageToIds = make(map[int][]string)
	t.topIds = make([]string, 0)
	t.dirty = false

	log.Printf("Cleared all top IDs data")
}

// GetStats returns statistics about the manager state
func (t *TopIdsManager) GetStats() (int, int, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.pageToIds), len(t.topIds), t.dirty
}
