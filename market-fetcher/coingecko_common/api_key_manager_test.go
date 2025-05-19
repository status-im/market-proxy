package coingecko_common

import (
	"testing"
	"time"

	"github.com/status-im/market-proxy/config"
)

// Helper function for tests to check if a key is in the list of available keys
func containsKey(keys []APIKey, key string, keyType KeyType) bool {
	for _, k := range keys {
		if k.Key == key && k.Type == keyType {
			return true
		}
	}
	return false
}

func TestAPIKeyManager_GetAvailableKeys(t *testing.T) {
	// Create test API tokens
	apiTokens := &config.APITokens{
		Tokens:     []string{"pro1", "pro2"},
		DemoTokens: []string{"demo1", "demo2", "demo3"},
	}

	// Create API key manager
	manager := NewAPIKeyManager(apiTokens)

	// Initially all keys should be available
	availableKeys := manager.GetAvailableKeys()

	// We expect 5 available keys (2 pro, 3 demo) plus NoKey
	if len(availableKeys) != 6 {
		t.Errorf("Expected 6 available keys (including NoKey), got %d", len(availableKeys))
	}

	// Verify pro and demo keys are present
	if !containsKey(availableKeys, "pro1", ProKey) {
		t.Errorf("Expected pro1 to be available")
	}
	if !containsKey(availableKeys, "pro2", ProKey) {
		t.Errorf("Expected pro2 to be available")
	}
	if !containsKey(availableKeys, "demo1", DemoKey) {
		t.Errorf("Expected demo1 to be available")
	}

	// Verify NoKey is present at the end
	lastKey := availableKeys[len(availableKeys)-1]
	if lastKey.Key != "" || lastKey.Type != NoKey {
		t.Errorf("Expected NoKey to be the last key in the list, got %v", lastKey)
	}

	// Mark one key as failed
	manager.MarkKeyAsFailed("pro1")

	// Now we should have one less available pro key, but still have 6 total keys
	availableKeys = manager.GetAvailableKeys()
	if len(availableKeys) != 5 {
		t.Errorf("Expected 5 available keys after marking one as failed, got %d", len(availableKeys))
	}

	// pro1 should no longer be in the list
	if containsKey(availableKeys, "pro1", ProKey) {
		t.Errorf("Expected pro1 to not be available after marking as failed")
	}

	// Test the special case: single Pro key should always be available
	// Create a manager with only one pro key
	singleProTokens := &config.APITokens{
		Tokens:     []string{"solo-pro"},
		DemoTokens: []string{"demo1", "demo2"},
	}
	singleProManager := NewAPIKeyManager(singleProTokens)

	// Mark the pro key as failed
	singleProManager.MarkKeyAsFailed("solo-pro")

	// The pro key should still be available
	singleProKeys := singleProManager.GetAvailableKeys()
	if !containsKey(singleProKeys, "solo-pro", ProKey) {
		t.Errorf("Expected solo pro key to be available even when in backoff")
	}
}

func TestAPIKeyManager_MarkKeyAsFailed(t *testing.T) {
	// Create test API tokens
	apiTokens := &config.APITokens{
		Tokens: []string{"pro1", "pro2", "pro3", "pro4"},
	}

	// Create API key manager with a shorter backoff for testing
	manager := NewAPIKeyManager(apiTokens)
	manager.backoffTime = 100 * time.Millisecond

	// Get initial available keys
	initialKeys := manager.GetAvailableKeys()
	initialProCount := 0
	for _, key := range initialKeys {
		if key.Type == ProKey {
			initialProCount++
		}
	}

	// Mark key as failed
	manager.MarkKeyAsFailed("pro1")

	// Get available keys after marking one as failed
	afterFailKeys := manager.GetAvailableKeys()
	afterFailProCount := 0
	for _, key := range afterFailKeys {
		if key.Type == ProKey {
			afterFailProCount++
		}
	}

	// We should have one less Pro key
	if afterFailProCount != (initialProCount - 1) {
		t.Errorf("Expected %d Pro keys after marking one as failed, got %d", initialProCount-1, afterFailProCount)
	}

	// Wait for backoff to expire
	time.Sleep(150 * time.Millisecond)

	// Get available keys after backoff expired
	afterBackoffKeys := manager.GetAvailableKeys()
	afterBackoffProCount := 0
	for _, key := range afterBackoffKeys {
		if key.Type == ProKey {
			afterBackoffProCount++
		}
	}

	// We should have same number of Pro keys as initially
	if afterBackoffProCount != initialProCount {
		t.Errorf("Expected %d Pro keys after backoff expired, got %d", initialProCount, afterBackoffProCount)
	}
}
