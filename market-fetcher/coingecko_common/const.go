package coingecko_common

const (
	// MaxChunkStringLength represents the maximum total length of strings in a single chunk
	// This helps avoid HTTP 414 "Request-URI Too Large" errors by limiting URL length
	MaxChunkStringLength = 7500
)
