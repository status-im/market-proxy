package e2etest

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
)

// waitForTokenListData waits for token list data to be available for a specific platform
func waitForTokenListData(t *testing.T, env *TestEnv, platform string) {
	maxWait := 10 * time.Second
	pollInterval := 200 * time.Millisecond
	timeout := time.Now().Add(maxWait)

	for time.Now().Before(timeout) {
		resp, err := http.Get(env.ServerBaseURL + "/api/v1/token_lists/" + platform + "/all.json")
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				body, err2 := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err2 == nil {
					var tokenListResponse map[string]interface{}
					if json.Unmarshal(body, &tokenListResponse) == nil {
						if tokens, ok := tokenListResponse["tokens"].([]interface{}); ok {
							t.Logf("Token list data initialization completed for %s, found %d tokens", platform, len(tokens))
							return
						}
					}
				}
			} else {
				resp.Body.Close()
			}
		}
		time.Sleep(pollInterval)
	}
	t.Logf("Token list data polling timeout for platform %s", platform)
}

// waitForDataInitialization waits for services to initialize data
func waitForDataInitialization(t *testing.T, env *TestEnv) {
	// Wait until service contains data by polling for expected tokens
	// This is more reliable than a fixed time delay
	maxWait := 30 * time.Second
	pollInterval := 500 * time.Millisecond
	timeout := time.Now().Add(maxWait)

	marketsReady := false
	leaderboardReady := false

	for time.Now().Before(timeout) {
		// Test if the markets cache contains the expected test tokens
		if !marketsReady {
			resp, err := http.Get(env.ServerBaseURL + "/api/v1/coins/markets?ids=bitcoin,ethereum")
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					body, err2 := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err2 == nil {
						var data []interface{}
						if json.Unmarshal(body, &data) == nil && len(data) > 0 {
							marketsReady = true
							t.Logf("Markets data initialization completed, found %d tokens in cache", len(data))
						}
					}
				} else {
					resp.Body.Close()
				}
			}
		}

		// Test if the leaderboard cache has data
		if marketsReady && !leaderboardReady {
			resp, err := http.Get(env.ServerBaseURL + "/api/v1/leaderboard/markets")
			if err == nil {
				if resp.StatusCode == http.StatusOK {
					body, err2 := io.ReadAll(resp.Body)
					resp.Body.Close()
					if err2 == nil {
						var leaderboardResponse map[string]interface{}
						if json.Unmarshal(body, &leaderboardResponse) == nil {
							if data, ok := leaderboardResponse["data"].([]interface{}); ok && len(data) > 0 {
								leaderboardReady = true
								t.Logf("Leaderboard data initialization completed, found %d tokens", len(data))
							}
						}
					}
				} else {
					resp.Body.Close()
				}
			}
		}

		// Both services are ready
		if marketsReady && leaderboardReady {
			t.Log("All services data initialization completed")
			return
		}

		time.Sleep(pollInterval)
	}

	// Fallback to original behavior if polling doesn't work
	t.Log("Data polling timeout, falling back to fixed wait time")
	if marketsReady {
		t.Log("Markets service ready, but leaderboard service not ready yet")
	}
	time.Sleep(2 * time.Second)
}

