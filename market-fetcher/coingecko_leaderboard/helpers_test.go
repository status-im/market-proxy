package coingecko_leaderboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// floatPtr builds the optional-value pointers the leaderboard response structs
// use. nil in those fields means "upstream did not report this", so tests
// distinguish floatPtr(0) from nil deliberately.
func floatPtr(value float64) *float64 {
	return &value
}

// requireFloat dereferences an optional response field, failing the test when
// it is absent
func requireFloat(t *testing.T, value *float64) float64 {
	t.Helper()
	require.NotNil(t, value, "expected the field to be present")
	return *value
}
