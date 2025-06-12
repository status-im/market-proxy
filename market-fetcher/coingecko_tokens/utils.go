package coingecko_tokens

// FilterTokensByPlatform filters tokens to keep only the supported platforms
// It returns a new slice of tokens with only the supported platforms in each token's platforms map
// Tokens without any supported platforms are excluded from the result
func FilterTokensByPlatform(tokens []Token, supportedPlatforms []string) []Token {
	result := make([]Token, 0, len(tokens))

	// Create a map for faster lookups
	supportedPlatformsMap := make(map[string]bool)
	for _, platform := range supportedPlatforms {
		supportedPlatformsMap[platform] = true
	}

	for _, token := range tokens {
		// Create a new platforms map with only supported platforms
		filteredPlatforms := make(map[string]string)

		for platform, address := range token.Platforms {
			if supportedPlatformsMap[platform] {
				filteredPlatforms[platform] = address
			}
		}

		// Only include tokens that have at least one supported platform
		if len(filteredPlatforms) > 0 {
			token.Platforms = filteredPlatforms
			result = append(result, token)
		}
	}

	return result
}

// CountTokensByPlatform counts the number of tokens per platform
func CountTokensByPlatform(tokens []Token) map[string]int {
	platformCounts := make(map[string]int)

	for _, token := range tokens {
		for platform := range token.Platforms {
			platformCounts[platform]++
		}
	}

	return platformCounts
}
