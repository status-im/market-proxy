# Currency conversion is a realtime Estimate computed from reference-coin ratios

The leaderboard endpoints serve prices in one configured currency (usd), but Status clients can display any currency ([status-app#21273](https://github.com/status-im/status-app/issues/21273)). Instead of fetching every currency from CoinGecko (multiplies API credit spend by the number of currencies) or mutating cached responses, we convert on request: `?convert_currency=X` on `/v1/leaderboard/markets` and `/v1/leaderboard/prices` returns an Estimate computed at request time from cached USD Passthrough values and a Ratio. Passthrough data is never mutated in cache or storage; Estimates are only ever realtime (or, if persisted later, in separate tables). The existing `currency` parameter keeps its passthrough-selection semantics; overloading it would silently switch between Passthrough and Estimate.

## Ratios via a reference coin, not `/exchange_rates`

Ratios (spot and 24h-ago) come from a single 1-credit call: `simple/price?ids=bitcoin,ethereum&vs_currencies=<configured list>&include_24hr_change=true&precision=full`, refreshed every minute. The reference coin's own price cancels out in the division, so its choice does not affect the Ratio — verified empirically: ratios derived via bitcoin, ethereum, and usd-coin agree to floating-point precision, and also match `/exchange_rates` for fiat. We rejected `/exchange_rates` as the Ratio source because it carries no 24h change (honest percent-change conversion would then require accumulating our own ratio history, with a cold-start gap after restarts) and its snapshot proved staler than simple/price. `/exchange_rates` is still exposed as a separate Passthrough endpoint, but conversion does not depend on it.

## Amendment: `convert_currency` names a currency, not a computation

The original rule above said Estimates are served only under an explicit URL marker, and that a currency named in both `vs_currencies` and `convert_currency` on `/v1/simple/price` is a 400 because one key cannot be both Passthrough and Estimate. That was wrong, and it showed up in the client: to phrase a legal request, status-go had to carry a hardcoded `proxyPassthroughCurrencies = {usd, eur, btc, eth}` mirroring `coingecko_prices.currencies` from this repo's `config.yaml`. Editing our config would silently break the client, and the client was making a decision it had no information for.

`?convert_currency=X` now means "give me values in X". The proxy picks the source: if it already holds provider values for X it serves those, because provider data beats anything we derive; otherwise it computes them as before. Naming the same currency in both parameters is a duplicate, not a conflict - it is deduped and served once. An unsupported currency is still a 400. `convert_currency` alone is now a complete request; `vs_currencies` is no longer required alongside it.

This only changes behaviour on `/v1/simple/price`, since it is the only endpoint whose cache holds more than one currency. The leaderboard caches the single currency in `coingecko_leaderboard.currency` and `/v1/coins/markets` is normalized to `market_params_normalize.vs_currency`, so a request for anything else there is still converted - but both now serve their own cached currency directly when asked for it, instead of computing a copy of data they already have.

The invariant survives: Passthrough values are still never mutated in cache or storage, and computed values are still realtime-only. What is gone is the promise that the URL tells you which of the two you got. That signal moves into the response as the `X-Estimated-Currencies` header, listing the currencies whose values the proxy computed; absent means everything served was provider data. Losing the URL-level marker is a real cost for debugging and for the test scripts, and the header is what pays it back.

## Consequences

- `percent_change_24h` is converted honestly for all target currencies (including btc/eth) via `(1 + pct_usd) × ratio_now / ratio_24h − 1`; a spot-ratio-only conversion would be wrong (e.g. bitcoin's change in btc terms must be ~0).
- Piggybacking the ratio fetch onto existing price calls is impossible: `vs_currencies` applies to every id in a call, so adding ~60 currencies would inflate the hot 500-token request 60×.
- An unknown `convert_currency` returns HTTP 400 — never a silent fallback to usd, which would serve one currency's values under another's name.
- On upstream failure the last known Ratio is served indefinitely (snapshot age exported as a metric) rather than failing the whole tab.
