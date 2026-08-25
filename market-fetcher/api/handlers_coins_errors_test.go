package api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/status-im/market-proxy/coingecko_coins"
	"github.com/status-im/market-proxy/coingecko_market_chart"
	"github.com/status-im/market-proxy/fetcher_by_id"
	"github.com/stretchr/testify/assert"
)

// TestErrNotFound_MatchesThroughWrapping pins the condition the coins handler
// branches on. The message stays what it was, so the 404 body is unchanged.
func TestErrNotFound_MatchesThroughWrapping(t *testing.T) {
	err := fmt.Errorf("%w: %s", fetcher_by_id.ErrNotFound, "bitcoin")

	assert.True(t, errors.Is(err, fetcher_by_id.ErrNotFound))
	assert.True(t, errors.Is(err, coingecko_coins.ErrNotFound),
		"the re-exported sentinel is the same value")
	assert.Equal(t, "item not found: bitcoin", err.Error(),
		"the message is unchanged, so the response body is too")
}

// TestErrNotFound_DoesNotMatchUnrelatedErrors is the reason for the change: the
// old strings.Contains check reclassified any error whose wording happened to
// include the same words.
func TestErrNotFound_DoesNotMatchUnrelatedErrors(t *testing.T) {
	for _, err := range []error{
		errors.New("failed to get from cache: key not found in backing store"),
		errors.New("upstream provider not found at address"),
	} {
		assert.Contains(t, err.Error(), "not found", "the old check would have matched this")
		assert.False(t, errors.Is(err, coingecko_coins.ErrNotFound),
			"a cache failure is a 500, not a 404: %v", err)
	}
}

func TestErrInvalidParams_MatchesThroughWrapping(t *testing.T) {
	err := fmt.Errorf("%w: %w", coingecko_market_chart.ErrInvalidParams, errors.New("days must be positive"))

	assert.True(t, errors.Is(err, coingecko_market_chart.ErrInvalidParams))
	assert.Equal(t, "invalid parameters: days must be positive", err.Error())

	assert.False(t, errors.Is(errors.New("data not found in cache"), coingecko_market_chart.ErrInvalidParams),
		"an internal cache miss stays a 500")
}
