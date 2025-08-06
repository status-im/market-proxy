package coingecko_markets

import (
	"fmt"
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
