# Market Proxy

Caching proxy between Status clients and market data providers (CoinGecko). Serves provider-compatible APIs plus custom aggregated endpoints (leaderboard).

## Language

**Passthrough**:
Data whose *values* are exactly as the upstream provider (CoinGecko) returned them. Passthrough values are never mutated in cache or storage, and are always preferred over an Estimate of the same thing.
_Avoid_: raw data, original data

**Estimate**:
A value computed by the proxy itself rather than returned by the provider — e.g. a price converted to another currency via a ratio. Estimates are computed at request time (realtime) or stored in tables separate from passthrough. Which of the two a response carries is the proxy's choice, not the client's: `?convert_currency=X` asks for values in X, and passthrough is served whenever the proxy already holds X. The response names any estimated currencies in the `X-Estimated-Currencies` header.
_Avoid_: converted data, derived data

**Ratio**:
An exchange rate between two quote currencies (e.g. usd→eur) used to compute Estimates from Passthrough prices. Derived from the provider's multi-currency price data for a reference coin; the reference coin's own price cancels out, so its choice does not affect the Ratio. Two ratios exist per currency: spot and 24h-ago (for honest percent-change conversion).
_Avoid_: rate, fx rate

**Leaderboard**:
The proxy's own aggregated API over top-N tokens: `/v1/leaderboard/markets` (full market rows) and `/v1/leaderboard/prices` (quote updates). Consumed by status-go's wallet leaderboard service via ETag polling.
