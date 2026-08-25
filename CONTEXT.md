# Market Proxy

Caching proxy between Status clients and market data providers (CoinGecko). Serves provider-compatible APIs plus custom aggregated endpoints (leaderboard).

## Language

**Passthrough**:
Data whose *values* are exactly as the upstream provider (CoinGecko) returned them. CoinGecko-compatible endpoints serve only passthrough. Passthrough values are never mutated in cache or storage.
_Avoid_: raw data, original data

**Estimate**:
A value computed by the proxy itself rather than returned by the provider — e.g. a price converted to another currency via a ratio. Estimates are served only under an explicit URL marker, and are either computed at request time (realtime) or stored in tables separate from passthrough.
_Avoid_: converted data, derived data

**Ratio**:
An exchange rate between two quote currencies (e.g. usd→eur) used to compute Estimates from Passthrough prices. Derived from the provider's multi-currency price data for a reference coin; the reference coin's own price cancels out, so its choice does not affect the Ratio. Two ratios exist per currency: spot and 24h-ago (for honest percent-change conversion).
_Avoid_: rate, fx rate

**Leaderboard**:
The proxy's own aggregated API over top-N tokens: `/v1/leaderboard/markets` (full market rows) and `/v1/leaderboard/prices` (quote updates). Consumed by status-go's wallet leaderboard service via ETag polling.
