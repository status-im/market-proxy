package fetcher_by_id

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	cg "github.com/status-im/market-proxy/coingecko_common"
	"github.com/status-im/market-proxy/config"
)

const (
	templatePlaceholderID      = "{{id}}"
	templatePlaceholderIDsList = "{{ids_list}}"
)

// RequestBuilder builds HTTP requests for fetcher_by_id endpoints
// It embeds CoingeckoRequestBuilder for consistent headers and userAgent handling
type RequestBuilder struct {
	*cg.CoingeckoRequestBuilder
	baseURL      string
	endpointPath string
	queryParams  map[string]string
	apiKey       string
	keyType      cg.KeyType
}

func NewRequestBuilder(baseURL string, cfg *config.FetcherByIdConfig) *RequestBuilder {
	return &RequestBuilder{
		CoingeckoRequestBuilder: cg.NewCoingeckoRequestBuilder(baseURL, ""),
		baseURL:                 strings.TrimRight(baseURL, "/"),
		endpointPath:            cfg.EndpointPath,
		queryParams:             cfg.BuildQueryParams(),
	}
}

func (rb *RequestBuilder) WithAPIKey(apiKey string, keyType cg.KeyType) *RequestBuilder {
	rb.apiKey = apiKey
	rb.keyType = keyType
	rb.CoingeckoRequestBuilder.WithApiKey(apiKey, keyType)
	return rb
}

func (rb *RequestBuilder) BuildSingleRequest(id string) (*http.Request, error) {
	urlStr, err := rb.BuildSingleURL(id)
	if err != nil {
		return nil, err
	}

	return rb.CoingeckoRequestBuilder.BuildWithURL(urlStr)
}

func (rb *RequestBuilder) BuildBatchRequest(ids []string) (*http.Request, error) {
	urlStr, err := rb.BuildBatchURL(ids)
	if err != nil {
		return nil, err
	}

	return rb.CoingeckoRequestBuilder.BuildWithURL(urlStr)
}

func (rb *RequestBuilder) BuildSingleURL(id string) (string, error) {
	if !strings.Contains(rb.endpointPath, templatePlaceholderID) {
		return "", fmt.Errorf("endpoint path does not contain %s placeholder", templatePlaceholderID)
	}

	path := strings.Replace(rb.endpointPath, templatePlaceholderID, url.PathEscape(id), 1)

	return rb.buildFinalURL(path)
}

func (rb *RequestBuilder) BuildBatchURL(ids []string) (string, error) {
	if !strings.Contains(rb.endpointPath, templatePlaceholderIDsList) {
		return "", fmt.Errorf("endpoint path does not contain %s placeholder", templatePlaceholderIDsList)
	}

	idsList := strings.Join(ids, ",")
	path := strings.Replace(rb.endpointPath, templatePlaceholderIDsList, idsList, 1)

	return rb.buildFinalURL(path)
}

func (rb *RequestBuilder) buildFinalURL(path string) (string, error) {
	parsedPath, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("failed to parse path: %w", err)
	}

	fullURL := rb.baseURL + parsedPath.Path
	query := parsedPath.Query()

	for key, value := range rb.queryParams {
		query.Set(key, value)
	}

	if rb.apiKey != "" {
		switch rb.keyType {
		case cg.ProKey:
			query.Set("x_cg_pro_api_key", rb.apiKey)
		case cg.DemoKey:
			query.Set("x_cg_demo_api_key", rb.apiKey)
		}
	}

	if len(query) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, query.Encode())
	}

	return fullURL, nil
}
