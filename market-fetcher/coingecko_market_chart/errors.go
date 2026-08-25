package coingecko_market_chart

import "errors"

// ErrInvalidParams is returned when the request parameters do not describe a
// chart that can be fetched.
//
// It exists so the HTTP layer can map the condition to 400 with errors.Is
// rather than matching the message text.
var ErrInvalidParams = errors.New("invalid parameters")
