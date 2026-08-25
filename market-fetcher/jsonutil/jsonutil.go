// Package jsonutil holds small helpers for reading typed values out of JSON
// that was decoded into interface{} / map[string]interface{}.
//
// Most upstream payloads are handled as generic maps rather than structs, so
// the same "is this field there and is it a number" question comes up in every
// service. These helpers answer it in one place, and always with an ok flag:
// a missing field and a genuine zero are different things, and only the caller
// knows whether collapsing them is acceptable.
package jsonutil

import "encoding/json"

// Float returns the float64 value of a decoded JSON value.
//
// ok is false for a missing key (nil), an explicit JSON null, and any
// non-numeric value. json.Number is accepted so that callers decoding with
// Decoder.UseNumber get the same answer as callers decoding into interface{}.
func Float(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case json.Number:
		number, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return number, true
	default:
		return 0, false
	}
}

// FloatField returns the float64 value of one field of a decoded JSON object.
// ok is false when the field is absent, null or not numeric.
func FloatField(object map[string]interface{}, key string) (float64, bool) {
	return Float(object[key])
}

// String returns the string value of a decoded JSON value.
// ok is false for a missing key, an explicit JSON null, and any non-string value.
func String(value interface{}) (string, bool) {
	str, ok := value.(string)
	return str, ok
}

// StringField returns the string value of one field of a decoded JSON object.
// ok is false when the field is absent, null or not a string.
func StringField(object map[string]interface{}, key string) (string, bool) {
	return String(object[key])
}
