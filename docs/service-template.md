# Adding a service

Every fetcher in `market-fetcher/` is built the same way. This walks the layers
in the order you will touch them, with real files to copy from.

Two references throughout:

- **`coingecko_assets_platforms`** — the minimal shape. No cache, no scheduler,
  one endpoint proxied on demand.
- **`currency_ratios`** — a recent full one. Config block, scheduler, in-memory
  snapshot, metrics, its own handler.

Skip the layers you do not need. A pure passthrough needs no scheduler; a
service nothing else consumes needs no `interfaces/` entry.

---

## 1. Config

Add the block to `market-fetcher/config.yaml`:

```yaml
currency_ratios:
  update_interval: 1m
  reference_coins: [bitcoin, ethereum]
```

Add the struct in `market-fetcher/config/<service>_config.go` — one file per
service, see `config/currency_ratios_config.go` or
`config/coingecko_exchange_rates_config.go`:

```go
type CurrencyRatiosConfig struct {
    UpdateInterval time.Duration `yaml:"update_interval"`
    ReferenceCoins []string      `yaml:"reference_coins"`
}

func GetDefaultCurrencyRatiosConfig() CurrencyRatiosConfig { ... }
func (c *CurrencyRatiosConfig) Validate() error { ... }
```

Register it in `config/config.go`: add the field to `Config`, apply the defaults
when the block is absent, and call `Validate()` from `LoadConfig`. A service that
silently runs on zero values is worse than one that refuses to start.

## 2. API client

`<service>/api.go` (`coingecko_assets_platforms` calls it `service.go`, which is
the odd one out — prefer `api.go`) holds the client and the interface the service
depends on:

```go
//go:generate mockgen -destination=mocks/client.go . IClient

type IClient interface {
    FetchReferencePrices(...) (SimplePricePayload, error)
    Healthy() bool
}
```

The client is boilerplate; copy `currency_ratios/api.go`:

- `cg.NewAPIKeyManager(cfg.APITokens)` and `cg.NewHTTPClientWithRetries(...)`
- `cg.GetApiBaseUrl(c.config, apiKey.Type)` for the pro/public split
- `cg.TryWithKeys(availableKeys, logPrefix, executor, onFailed)` for key rotation
- an `atomic.Bool` set on the first success, returned by `Healthy()`

`<service>/<service>_request_builder.go` wraps `cg.NewCoingeckoRequestBuilder`
with `With*` methods — see `currency_ratios/ratios_request_builder.go`. Keep the
CoinGecko path in a constant next to it.

## 3. Service

`<service>/service.go` implements `core.IService` (`Start(ctx) error`, `Stop()`)
plus `Healthy() bool`:

```go
func NewService(cfg *config.Config) *Service {
    metricsWriter := metrics.NewMetricsWriter(metrics.ServiceCurrencyRatios)
    return NewServiceWithClient(cfg.CurrencyRatios, NewCoinGeckoClient(cfg, metricsWriter), metricsWriter)
}
```

The second constructor taking an explicit client is what makes the service
testable without HTTP. Do it in both new services.

Periodic work uses `scheduler.New(interval, task)` and
`scheduler.Start(ctx, true)` — the `true` runs the first fetch immediately
instead of waiting out the interval. Guard `interval <= 0` and log that updates
are disabled rather than starting a hot loop.

In the fetch cycle:

```go
s.metricsWriter.ResetCycleMetrics()
defer s.metricsWriter.TrackDataFetchCycle()()
```

and `RecordCacheSize` after a successful update. **On a failed fetch, keep the
previous data** and return the error to be logged — see
`docs/adr/0001-currency-conversion-as-realtime-estimate.md`. Serving stale data
beats failing the tab, as long as the staleness is visible as a metric.

For a shared cache use `cache.ICache` (`coingecko_markets`, `coingecko_prices`);
for a single in-memory snapshot an `atomic.Pointer[T]` is enough
(`currency_ratios`, `coingecko_exchange_rates`).

## 4. Wiring

`core/composition_root.go`, in dependency order:

```go
svc := <service>.NewService(cfg)
registry.Register(svc)
```

Registration order is start order, and `StopAll` runs it in reverse. Pass the
service into `api.New(...)` at the bottom.

## 5. HTTP

`api/handlers_<service>.go` — one file per service. Handlers **only** parse and
validate request params, call the service, and map the result to a status code.
Anything that transforms data belongs in the service; see the `convert_currency`
handlers for the shape.

Route it in `api/server.go` under `/api/v1/...`, and use `s.sendJSONResponse`
(sets Content-Type, Content-Length and the ETag that status-go polls on) or
`s.sendJSONError` for a JSON error body.

If another package consumes the service, add `interfaces/<service>.go` with the
interface, its DTOs and a `//go:generate mockgen` directive, then run
`go generate ./interfaces/...`. Embed `IHealthReporter` if `/health` reports it.
Interfaces used only inside their own package (API clients, sub-components) stay
in the package.

## 6. nginx

`nginx-proxy/nginx.conf` — a `proxy_cache_path ... keys_zone=<service>_cache`
next to the others, and a location block:

```nginx
location = /v1/exchange_rates {
    proxy_pass http://market-fetcher:8081/api/v1/exchange_rates;
    proxy_cache exchange_rates_cache;
    proxy_cache_key "$request_uri";
    proxy_cache_valid 200 60s;
    proxy_cache_valid 304 60s;
    proxy_cache_use_stale error timeout http_500 http_502 http_503 http_504;
    add_header X-Cache-Status $upstream_cache_status always;
    add_header X-Proxy-Cache $upstream_cache_status always;
}
```

`proxy_cache_key "$request_uri"` includes the query string, so query params vary
the cache entry without extra config. Add `auth_basic` only if the endpoint is
one of the proxy's own APIs — the CoinGecko-compatible ones are public.

Add the same location to `nginx-proxy/patch-local-cors.sh` so it gets CORS
headers in local development.

## 7. Tests

- **Unit tests next to the package.** `service_test.go` with a stub client
  (`currency_ratios/service_test.go`) and `api_test.go` for the client and
  request builder (`coingecko_assets_platforms/api_test.go`). Cross-package
  dependencies use the generated mocks from `interfaces/mocks/`.
- **`e2etest/`** boots the whole registry against `e2etest/http_mock.go`. Add a
  branch there returning your fixture, and a `<service>_test.go` hitting the real
  HTTP endpoint. Existing branches match on path; if yours collides with another
  service's path, discriminate on a query param the way the `precision=full`
  branch does for the ratios `simple/price` call.

---

## Don't forget

- [ ] `metrics.Service<Name>` constant in `metrics/service_metrics.go`, and a
      `MetricsWriter` passed into the client — without it the service's upstream
      calls are invisible in Grafana. New metric vectors need a cardinality note
      in the comment, like the existing ones.
- [ ] `api_test.go` covering the client. Most services have one;
      `currency_ratios` was merged without it and had to be backfilled.
- [ ] `Validate()` on the config, called from `LoadConfig`.
- [ ] Failed fetch keeps the previous data, and staleness is observable.
- [ ] `README.md` (repo root, endpoint list) and `market-fetcher/README.md`
      (endpoint documentation with a sample response).
- [ ] `test-api/src/components/EndpointTester.js` — add the endpoint so the
      browser tester exercises it.
- [ ] `market-proxy-scripts/` (separate repo) — `test-proxy-endpoints.py`,
      the endpoint list in `scripts.md`, and `requests.ndjson`.
- [ ] `go vet ./...` and `go test ./...` green; `gofmt -l .` clean.
