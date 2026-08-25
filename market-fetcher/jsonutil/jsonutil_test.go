package jsonutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFloat(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected float64
		ok       bool
	}{
		{name: "float64", value: 1.5, expected: 1.5, ok: true},
		{name: "zero", value: 0.0, expected: 0, ok: true},
		{name: "negative", value: -2.25, expected: -2.25, ok: true},
		{name: "json.Number", value: json.Number("2.5"), expected: 2.5, ok: true},
		{name: "json.Number integer", value: json.Number("7"), expected: 7, ok: true},
		{name: "invalid json.Number", value: json.Number("abc"), ok: false},
		{name: "nil (json null or missing key)", value: nil, ok: false},
		{name: "string", value: "1.5", ok: false},
		{name: "bool", value: true, ok: false},
		{name: "int is not produced by encoding/json", value: 3, ok: false},
		{name: "nested object", value: map[string]interface{}{}, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Float(tt.value)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.expected, got)
			} else {
				assert.Zero(t, got)
			}
		})
	}
}

// TestFloatField_DistinguishesMissingFromZero is the reason the helper carries
// an ok flag at all.
func TestFloatField_DistinguishesMissingFromZero(t *testing.T) {
	var object map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(`{"price": 0, "volume": null}`), &object))

	price, ok := FloatField(object, "price")
	assert.True(t, ok, "a real zero is a value")
	assert.Equal(t, 0.0, price)

	_, ok = FloatField(object, "volume")
	assert.False(t, ok, "an explicit null is not a value")

	_, ok = FloatField(object, "market_cap")
	assert.False(t, ok, "an absent key is not a value")
}

func TestFloatField_NilMap(t *testing.T) {
	_, ok := FloatField(nil, "price")
	assert.False(t, ok)
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
		ok       bool
	}{
		{name: "string", value: "bitcoin", expected: "bitcoin", ok: true},
		{name: "empty string is a value", value: "", expected: "", ok: true},
		{name: "nil (json null or missing key)", value: nil, ok: false},
		{name: "number", value: 1.0, ok: false},
		{name: "bool", value: false, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := String(tt.value)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestStringField(t *testing.T) {
	var object map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(`{"id": "bitcoin", "name": null}`), &object))

	id, ok := StringField(object, "id")
	assert.True(t, ok)
	assert.Equal(t, "bitcoin", id)

	_, ok = StringField(object, "name")
	assert.False(t, ok)

	_, ok = StringField(object, "symbol")
	assert.False(t, ok)
}
