package coingecko_tokens

import "github.com/status-im/market-proxy/interfaces"

// FilterTokensByPlatform filters tokens to keep only supported platforms and native tokens
func FilterTokensByPlatform(tokens []interfaces.Token, supportedPlatforms []string) []interfaces.Token {
	result := make([]interfaces.Token, 0, len(tokens))

	// Create a map for faster lookups
	supportedPlatformsMap := make(map[string]bool)
	for _, platform := range supportedPlatforms {
		supportedPlatformsMap[platform] = true
	}

	for _, token := range tokens {
		// Check if token ID is a supported platform (native token)
		isNativeToken := supportedPlatformsMap[token.ID]

		// Filter platforms to keep only supported ones
		filteredPlatforms := make(map[string]string)
		for platform, address := range token.Platforms {
			if supportedPlatformsMap[platform] {
				filteredPlatforms[platform] = address
			}
		}

		// Include token if it's a native token OR has supported platforms
		if isNativeToken || len(filteredPlatforms) > 0 {
			token.Platforms = filteredPlatforms
			result = append(result, token)
		}
	}

	return result
}

// CountTokensByPlatform counts tokens per platform
func CountTokensByPlatform(tokens []interfaces.Token) map[string]int {
	platformCounts := make(map[string]int)

	for _, token := range tokens {
		for platform := range token.Platforms {
			platformCounts[platform]++
		}
	}

	return platformCounts
}
