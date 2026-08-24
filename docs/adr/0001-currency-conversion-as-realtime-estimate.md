# Currency conversion is a realtime Estimate computed from reference-coin ratios

The leaderboard endpoints serve prices in one configured currency (usd), but Status clients can display any currency ([status-app#21273](https://github.com/status-im/status-app/issues/21273)). Instead of fetching every currency from CoinGecko (multiplies API credit spend by the number of currencies) or mutating cached responses, we convert on request: `?convert_currency=X` on `/v1/leaderboard/markets` and `/v1/leaderboard/prices` returns an Estimate computed at request time from cached USD Passthrough values and a Ratio. Passthrough data is never mutated in cache or storage; Estimates are only ever realtime (or, if persisted later, in separate tables). The existing `currency` parameter keeps its passthrough-selection semantics; overloading it would silently switch between Passthrough and Estimate.

## Ratios via a reference coin, not `/exchange_rates`

Ratios (spot and 24h-ago) come from a single 1-credit call: `simple/price?ids=bitcoin,ethereum&vs_currencies=<configured list>&include_24hr_change=true&precision=full`, refreshed every minute. The reference coin's own price cancels out in the division, so its choice does not affect the Ratio — verified empirically: ratios derived via bitcoin, ethereum, and usd-coin agree to floating-point precision, and also match `/exchange_rates` for fiat. We rejected `/exchange_rates` as the Ratio source because it carries no 24h change (honest percent-change conversion would then require accumulating our own ratio history, with a cold-start gap after restarts) and its snapshot proved staler than simple/price. `/exchange_rates` is still exposed as a separate Passthrough endpoint, but conversion does not depend on it.

## Consequences

- `percent_change_24h` is converted honestly for all target currencies (including btc/eth) via `(1 + pct_usd) × ratio_now / ratio_24h − 1`; a spot-ratio-only conversion would be wrong (e.g. bitcoin's change in btc terms must be ~0).
- Piggybacking the ratio fetch onto existing price calls is impossible: `vs_currencies` applies to every id in a call, so adding ~60 currencies would inflate the hot 500-token request 60×.
- An unknown `convert_currency` returns HTTP 400 — never a silent fallback to usd, which would masquerade Passthrough as Estimate.
- On upstream failure the last known Ratio is served indefinitely (snapshot age exported as a metric) rather than failing the whole tab.
