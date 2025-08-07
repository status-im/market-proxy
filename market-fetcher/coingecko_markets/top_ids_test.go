package coingecko_markets

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTopIdsManager_UpdatePageIds(t *testing.T) {
	manager := NewTopIdsManager()

	t.Run("Update single page", func(t *testing.T) {
		tokenIds := []string{"bitcoin", "ethereum", "cardano"}
		manager.UpdatePageIds(1, tokenIds)

		result := manager.GetTopIds(0)
		assert.Equal(t, tokenIds, result)

		pageCount, totalTokens, isDirty := manager.GetStats()
		assert.Equal(t, 1, pageCount)
		assert.Equal(t, 3, totalTokens)
		assert.False(t, isDirty) // Should be false after GetTopIds call
	})

	t.Run("Update multiple pages", func(t *testing.T) {
		manager.Clear()

		// Add page 1
		page1Tokens := []string{"bitcoin", "ethereum"}
		manager.UpdatePageIds(1, page1Tokens)

		// Add page 2
		page2Tokens := []string{"cardano", "solana"}
		manager.UpdatePageIds(2, page2Tokens)

		// Should maintain order
		expected := []string{"bitcoin", "ethereum", "cardano", "solana"}
		result := manager.GetTopIds(0)
		assert.Equal(t, expected, result)
	})

	t.Run("Update existing page", func(t *testing.T) {
		manager.Clear()

		// Initial page
		manager.UpdatePageIds(1, []string{"bitcoin", "ethereum"})

		// Update same page with different tokens
		manager.UpdatePageIds(1, []string{"cardano", "solana", "polkadot"})

		result := manager.GetTopIds(0)
		assert.Equal(t, []string{"cardano", "solana", "polkadot"}, result)
	})
}

func TestTopIdsManager_UpdatePagesFromPageData(t *testing.T) {
	manager := NewTopIdsManager()

	t.Run("Update from PageData", func(t *testing.T) {
		pagesData := []PageData{
			{
				Page: 1,
				Data: [][]byte{
					[]byte(`{"id":"bitcoin","symbol":"btc"}`),
					[]byte(`{"id":"ethereum","symbol":"eth"}`),
				},
			},
			{
				Page: 2,
				Data: [][]byte{
					[]byte(`{"id":"cardano","symbol":"ada"}`),
				},
			},
		}

		manager.UpdatePagesFromPageData(pagesData)

		expected := []string{"bitcoin", "ethereum", "cardano"}
		result := manager.GetTopIds(0)
		assert.Equal(t, expected, result)
	})

	t.Run("Handle invalid JSON", func(t *testing.T) {
		manager.Clear()

		pagesData := []PageData{
			{
				Page: 1,
				Data: [][]byte{
					[]byte(`{"id":"bitcoin","symbol":"btc"}`),
					[]byte(`invalid json`),
					[]byte(`{"id":"ethereum","symbol":"eth"}`),
				},
			},
		}

		manager.UpdatePagesFromPageData(pagesData)

		// Should only extract valid tokens
		expected := []string{"bitcoin", "ethereum"}
		result := manager.GetTopIds(0)
		assert.Equal(t, expected, result)
	})
}

func TestTopIdsManager_GetTopIds(t *testing.T) {
	manager := NewTopIdsManager()

	t.Run("Get with limit", func(t *testing.T) {
		tokens := []string{"bitcoin", "ethereum", "cardano", "solana", "polkadot"}
		manager.UpdatePageIds(1, tokens)

		// Test different limits
		result := manager.GetTopIds(3)
		assert.Equal(t, tokens[:3], result)

		result = manager.GetTopIds(0) // Should return all
		assert.Equal(t, tokens, result)

		result = manager.GetTopIds(10) // Should return all (limit > available)
		assert.Equal(t, tokens, result)
	})

	t.Run("Empty manager", func(t *testing.T) {
		emptyManager := NewTopIdsManager()
		result := emptyManager.GetTopIds(5)
		assert.Empty(t, result)
	})
}

func TestTopIdsManager_GetPageIds(t *testing.T) {
	manager := NewTopIdsManager()

	t.Run("Get existing page", func(t *testing.T) {
		tokens := []string{"bitcoin", "ethereum"}
		manager.UpdatePageIds(1, tokens)

		result := manager.GetPageIds(1)
		assert.Equal(t, tokens, result)
	})

	t.Run("Get non-existing page", func(t *testing.T) {
		result := manager.GetPageIds(999)
		assert.Empty(t, result)
	})
}

func TestTopIdsManager_GetAvailablePages(t *testing.T) {
	manager := NewTopIdsManager()

	t.Run("Multiple pages", func(t *testing.T) {
		manager.UpdatePageIds(1, []string{"bitcoin"})
		manager.UpdatePageIds(3, []string{"ethereum"})
		manager.UpdatePageIds(2, []string{"cardano"})

		pages := manager.GetAvailablePages()
		assert.Len(t, pages, 3)
		assert.Contains(t, pages, 1)
		assert.Contains(t, pages, 2)
		assert.Contains(t, pages, 3)
	})

	t.Run("Empty manager", func(t *testing.T) {
		emptyManager := NewTopIdsManager()
		pages := emptyManager.GetAvailablePages()
		assert.Empty(t, pages)
	})
}

func TestTopIdsManager_Clear(t *testing.T) {
	manager := NewTopIdsManager()

	// Add some data
	manager.UpdatePageIds(1, []string{"bitcoin", "ethereum"})
	manager.UpdatePageIds(2, []string{"cardano"})

	// Verify data exists
	assert.NotEmpty(t, manager.GetTopIds(0))
	assert.NotEmpty(t, manager.GetAvailablePages())

	// Clear
	manager.Clear()

	// Verify everything is cleared
	assert.Empty(t, manager.GetTopIds(0))
	assert.Empty(t, manager.GetAvailablePages())

	pageCount, totalTokens, isDirty := manager.GetStats()
	assert.Equal(t, 0, pageCount)
	assert.Equal(t, 0, totalTokens)
	assert.False(t, isDirty)
}

func TestTopIdsManager_Deduplication(t *testing.T) {
	manager := NewTopIdsManager()

	t.Run("Duplicates across pages are removed", func(t *testing.T) {
		manager.Clear()

		// Page 1: bitcoin, ethereum, cardano
		page1Tokens := []string{"bitcoin", "ethereum", "cardano"}
		manager.UpdatePageIds(1, page1Tokens)

		// Page 2: ethereum, cardano, solana (ethereum and cardano are duplicates)
		page2Tokens := []string{"ethereum", "cardano", "solana"}
		manager.UpdatePageIds(2, page2Tokens)

		// Page 3: cardano, solana, polkadot (cardano and solana are duplicates)
		page3Tokens := []string{"cardano", "solana", "polkadot"}
		manager.UpdatePageIds(3, page3Tokens)

		// Expected: bitcoin, ethereum, cardano, solana, polkadot (in order of first appearance)
		expected := []string{"bitcoin", "ethereum", "cardano", "solana", "polkadot"}
		result := manager.GetTopIds(0)
		assert.Equal(t, expected, result)

		// Check duplicate stats
		duplicates := manager.GetDuplicateStats()
		assert.Len(t, duplicates, 3) // ethereum, cardano, solana appear in multiple pages

		// Check specific duplicates
		assert.Contains(t, duplicates, "ethereum")
		assert.Contains(t, duplicates, "cardano")
		assert.Contains(t, duplicates, "solana")

		// Verify pages where duplicates appear (sort for consistent testing)
		ethereumPages := duplicates["ethereum"]
		sort.Ints(ethereumPages)
		assert.Equal(t, []int{1, 2}, ethereumPages)

		cardanoPages := duplicates["cardano"]
		sort.Ints(cardanoPages)
		assert.Equal(t, []int{1, 2, 3}, cardanoPages)

		solanaPages := duplicates["solana"]
		sort.Ints(solanaPages)
		assert.Equal(t, []int{2, 3}, solanaPages)
	})

	t.Run("No duplicates case", func(t *testing.T) {
		manager.Clear()

		// Page 1: unique tokens
		manager.UpdatePageIds(1, []string{"bitcoin", "ethereum"})
		// Page 2: different unique tokens
		manager.UpdatePageIds(2, []string{"cardano", "solana"})

		expected := []string{"bitcoin", "ethereum", "cardano", "solana"}
		result := manager.GetTopIds(0)
		assert.Equal(t, expected, result)

		// Should have no duplicates
		duplicates := manager.GetDuplicateStats()
		assert.Empty(t, duplicates)
	})

	t.Run("Token appears in non-consecutive pages", func(t *testing.T) {
		manager.Clear()

		// Page 1: bitcoin, ethereum
		manager.UpdatePageIds(1, []string{"bitcoin", "ethereum"})
		// Page 2: cardano, solana
		manager.UpdatePageIds(2, []string{"cardano", "solana"})
		// Page 4: bitcoin again (skipping page 3)
		manager.UpdatePageIds(4, []string{"bitcoin", "polkadot"})

		// Expected: bitcoin (from page 1), ethereum, cardano, solana, polkadot
		expected := []string{"bitcoin", "ethereum", "cardano", "solana", "polkadot"}
		result := manager.GetTopIds(0)
		assert.Equal(t, expected, result)

		// Check duplicates
		duplicates := manager.GetDuplicateStats()
		assert.Len(t, duplicates, 1)
		assert.Contains(t, duplicates, "bitcoin")

		bitcoinPages := duplicates["bitcoin"]
		sort.Ints(bitcoinPages)
		assert.Equal(t, []int{1, 4}, bitcoinPages)
	})

	t.Run("All tokens are duplicates", func(t *testing.T) {
		manager.Clear()

		// All pages have the same tokens
		sameTokens := []string{"bitcoin", "ethereum"}
		manager.UpdatePageIds(1, sameTokens)
		manager.UpdatePageIds(2, sameTokens)
		manager.UpdatePageIds(3, sameTokens)

		// Should only keep first occurrence
		result := manager.GetTopIds(0)
		assert.Equal(t, sameTokens, result)

		// All tokens should be duplicates
		duplicates := manager.GetDuplicateStats()
		assert.Len(t, duplicates, 2)

		bitcoinPages := duplicates["bitcoin"]
		sort.Ints(bitcoinPages)
		assert.Equal(t, []int{1, 2, 3}, bitcoinPages)

		ethereumPages := duplicates["ethereum"]
		sort.Ints(ethereumPages)
		assert.Equal(t, []int{1, 2, 3}, ethereumPages)
	})
}

func TestTopIdsManager_ConcurrentAccess(t *testing.T) {
	manager := NewTopIdsManager()

	// This test verifies that the mutex protects against race conditions
	t.Run("Concurrent updates and reads", func(t *testing.T) {
		done := make(chan bool, 10)

		// Start multiple goroutines updating different pages
		for i := 0; i < 5; i++ {
			go func(page int) {
				tokens := []string{fmt.Sprintf("token%d", page)}
				manager.UpdatePageIds(page, tokens)
				done <- true
			}(i + 1)
		}

		// Start multiple goroutines reading data
		for i := 0; i < 5; i++ {
			go func() {
				_ = manager.GetTopIds(10)
				_ = manager.GetAvailablePages()
				_, _, _ = manager.GetStats()
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify we have some data (exact count may vary due to concurrency)
		result := manager.GetTopIds(0)
		assert.True(t, len(result) <= 5) // Should have at most 5 tokens
		assert.True(t, len(result) > 0)  // Should have at least some tokens
	})
}
