package coingecko_markets

import (
	"log"
	"sync"
)

// TopIdsManager manages top token IDs with efficient page-based updates
type TopIdsManager struct {
	mu sync.RWMutex

	pageToIds map[int][]string // page number -> tokenIds
	topIds    []string         // top Ids (built gradually when receiving pages)
	dirty     bool             // need to rebuild
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
func (t *TopIdsManager) UpdatePageIds(page int, tokenIds []string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.pageToIds[page] = make([]string, len(tokenIds))
	copy(t.pageToIds[page], tokenIds)

	// Mark as dirty for rebuild
	t.dirty = true
}

// UpdatePagesFromPageData updates multiple pages from PageData slice
func (t *TopIdsManager) UpdatePagesFromPageData(pagesData []PageData) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, pageData := range pagesData {
		tokenIds := extractTokenIDsFromPageData(pageData.Data)

		t.pageToIds[pageData.Page] = make([]string, len(tokenIds))
		copy(t.pageToIds[pageData.Page], tokenIds)
	}

	// Mark as dirty for rebuild
	t.dirty = true
}

// GetTopIds returns the current top IDs vector up to the specified limit
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
		total := 0
		for _, ids := range t.pageToIds {
			total += len(ids)
		}
		return total
	}

	return len(t.topIds)
}

// rebuildTopIds rebuilds the topIds vector from page data with deduplication
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

	// Use a map to track seen IDs for deduplication
	seenIds := make(map[string]int) // tokenId -> first page seen
	var newTopIds []string

	// Rebuild topIds in page order, keeping only the first occurrence of each ID
	for page := 1; page <= maxPage; page++ {
		if ids, exists := t.pageToIds[page]; exists {
			for _, id := range ids {
				if _, seen := seenIds[id]; !seen {
					seenIds[id] = page
					newTopIds = append(newTopIds, id)
				}
			}
		}
	}

	t.topIds = newTopIds
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

// GetDuplicateStats returns information about duplicate tokens across pages
func (t *TopIdsManager) GetDuplicateStats() map[string][]int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Map token ID to pages where it appears
	tokenToPages := make(map[string][]int)

	for page, ids := range t.pageToIds {
		for _, id := range ids {
			tokenToPages[id] = append(tokenToPages[id], page)
		}
	}

	// Return only tokens that appear in multiple pages
	duplicates := make(map[string][]int)
	for tokenId, pages := range tokenToPages {
		if len(pages) > 1 {
			duplicates[tokenId] = pages
		}
	}

	return duplicates
}
