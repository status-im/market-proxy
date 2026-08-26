package coingecko_common

import (
	"github.com/status-im/market-proxy/config"
	"github.com/status-im/proxy-common/apikeys"
)

// tokensProvider adapts config.APITokens to apikeys.KeyProvider.
type tokensProvider struct{ t *config.APITokens }

func (p tokensProvider) GetKeys(kt apikeys.KeyType) []string {
	switch kt {
	case ProKey:
		if p.t == nil {
			return nil
		}
		return p.t.Tokens
	case DemoKey:
		if p.t == nil {
			return nil
		}
		return p.t.DemoTokens
	case NoKey:
		// len==1 rule in apikeys manager always includes the entry (even in "backoff")
		return []string{""}
	default:
		return nil
	}
}
