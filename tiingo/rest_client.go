package tiingo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/condrove10/retryablehttp"
	"github.com/condrove10/retryablehttp/backoffpolicy"
	"github.com/google/go-querystring/query"
	"golang.org/x/time/rate"
)

const (
	defaultBaseURL        = "https://api.tiingo.com"
	defaultRequestsPerSec = 50
	defaultRetryMax       = 3
	defaultRetryDelay     = 1 * time.Second
	defaultTimeout        = 30 * time.Second
)

const (
	StrategyLinear      = backoffpolicy.StrategyLinear
	StrategyExponential = backoffpolicy.StrategyExponential
)

// RestClient handles communication with the Tiingo REST API.
type RestClient struct {
	apiKey      string
	baseURL     string
	httpClient  *retryablehttp.Client
	rateLimiter *rate.Limiter
}

// RestConfig holds configuration for the REST client.
type RestConfig struct {
	BaseURL        string
	RequestsPerSec int
	RetryMax       uint32
	RetryDelay     time.Duration
	Timeout        time.Duration
	HTTPClient     *http.Client
	RetryPolicy    func(resp *http.Response, err error) error
	RetryStrategy  backoffpolicy.Strategy
}

// RestOption is a functional option for configuring the RestClient.
type RestOption func(*RestConfig)

// WithBaseURL sets a custom base URL.
func WithBaseURL(baseURL string) RestOption {
	return func(c *RestConfig) { c.BaseURL = baseURL }
}

// WithRequestsPerSecond sets the rate limit for requests.
func WithRequestsPerSecond(rps int) RestOption {
	return func(c *RestConfig) { c.RequestsPerSec = rps }
}

// WithRetryMax sets the maximum number of retry attempts.
func WithRetryMax(max uint32) RestOption {
	return func(c *RestConfig) { c.RetryMax = max }
}

// WithRetryDelay sets the delay between retry attempts.
func WithRetryDelay(delay time.Duration) RestOption {
	return func(c *RestConfig) { c.RetryDelay = delay }
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(timeout time.Duration) RestOption {
	return func(c *RestConfig) { c.Timeout = timeout }
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) RestOption {
	return func(c *RestConfig) { c.HTTPClient = client }
}

// WithRetryPolicy sets a custom retry policy.
func WithRetryPolicy(policy func(resp *http.Response, err error) error) RestOption {
	return func(c *RestConfig) { c.RetryPolicy = policy }
}

// WithStrategy sets the backoff strategy for retries.
func WithStrategy(strategy backoffpolicy.Strategy) RestOption {
	return func(c *RestConfig) { c.RetryStrategy = strategy }
}

// NewRestClient creates a new client for the REST API.
func NewRestClient(ctx context.Context, apiKey string, options ...RestOption) (*RestClient, error) {
	config := &RestConfig{
		BaseURL:        defaultBaseURL,
		RequestsPerSec: defaultRequestsPerSec,
		RetryMax:       defaultRetryMax,
		RetryDelay:     defaultRetryDelay,
		Timeout:        defaultTimeout,
		RetryStrategy:  StrategyLinear,
	}

	for _, option := range options {
		option(config)
	}

	retryOpts := []retryablehttp.ClientOption{
		retryablehttp.WithAttempts(config.RetryMax),
		retryablehttp.WithDelay(config.RetryDelay),
	}

	if config.HTTPClient != nil {
		retryOpts = append(retryOpts, retryablehttp.WithHttpClient(config.HTTPClient))
	} else if config.Timeout != defaultTimeout {
		customClient := &http.Client{
			Timeout: config.Timeout,
		}
		retryOpts = append(retryOpts, retryablehttp.WithHttpClient(customClient))
	}

	if config.RetryPolicy != nil {
		retryOpts = append(retryOpts, retryablehttp.WithPolicy(config.RetryPolicy))
	}

	httpClient, err := retryablehttp.New(ctx, retryOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create retryablehttp client: %w", err)
	}

	limiter := rate.NewLimiter(rate.Every(time.Second/time.Duration(config.RequestsPerSec)), config.RequestsPerSec)

	return &RestClient{
		apiKey:      apiKey,
		baseURL:     config.BaseURL,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}, nil
}

// buildURL constructs a full URL with token and optional query parameters.
func (c *RestClient) buildURL(path string, opts interface{}) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}
	u.Path = path

	v := url.Values{}
	if opts != nil {
		v, err = query.Values(opts)
		if err != nil {
			return "", fmt.Errorf("failed to encode options: %w", err)
		}
	}

	v.Set("token", c.apiKey)
	u.RawQuery = v.Encode()

	return u.String(), nil
}

// request executes a given request after waiting for the rate limiter and decodes the response.
func (c *RestClient) request(ctx context.Context, method, path string, body []byte, opts, result interface{}) error {
	url, err := c.buildURL(path, opts)
	if err != nil {
		return err
	}

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return err
	}

	headers := map[string]string{"Content-Type": "application/json"}
	resp, err := c.httpClient.Do(url, method, body, headers)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if err := handleHTTPError(resp); err != nil {
		return err
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response body: %w", err)
	}

	return nil
}

// GetTickerMetadata retrieves metadata for a given stock ticker.
func (c *RestClient) GetTickerMetadata(ctx context.Context, ticker string) (*TickerMetadata, error) {
	path := fmt.Sprintf("/tiingo/daily/%s", ticker)
	var metadata TickerMetadata
	err := c.request(ctx, http.MethodGet, path, nil, nil, &metadata)
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

// GetEndOfDayPrices retrieves end-of-day price data for a ticker.
func (c *RestClient) GetEndOfDayPrices(ctx context.Context, ticker string, opts *EndOfDayPricesOptions) ([]*PriceData, error) {
	path := fmt.Sprintf("/tiingo/daily/%s/prices", ticker)
	var prices []*PriceData
	err := c.request(ctx, http.MethodGet, path, nil, opts, &prices)
	if err != nil {
		return nil, err
	}
	return prices, nil
}

// GetIEXRealTimePrices retrieves IEX real-time price data.
func (c *RestClient) GetIEXRealTimePrices(ctx context.Context, opts *IEXRealTimePricesOptions) ([]*IEXQuote, error) {
	path := fmt.Sprintf("/iex/%s", opts.Tickers)
	opts.Tickers = ""
	var quotes []*IEXQuote
	err := c.request(ctx, http.MethodGet, path, nil, opts, &quotes)
	if err != nil {
		return nil, err
	}
	return quotes, nil
}

// GetCryptoPrices retrieves cryptocurrency price data.
func (c *RestClient) GetCryptoPrices(ctx context.Context, opts *CryptoPricesOptions) ([]*CryptoPrice, error) {
	path := "/tiingo/crypto/prices"
	var prices []*CryptoPrice
	err := c.request(ctx, http.MethodGet, path, nil, opts, &prices)
	if err != nil {
		return nil, err
	}
	return prices, nil
}

// GetForexPrices retrieves forex price data.
func (c *RestClient) GetForexPrices(ctx context.Context, ticker string, opts *ForexPricesOptions) ([]*PriceData, error) {
	path := fmt.Sprintf("/tiingo/fx/%s/prices", ticker)
	var prices []*PriceData
	err := c.request(ctx, http.MethodGet, path, nil, opts, &prices)
	if err != nil {
		return nil, err
	}
	return prices, nil
}

// GetNewsFeed retrieves news articles.
func (c *RestClient) GetNewsFeed(ctx context.Context, opts *NewsFeedOptions) ([]*NewsItem, error) {
	path := "/tiingo/news"
	var news []*NewsItem
	err := c.request(ctx, http.MethodGet, path, nil, opts, &news)
	if err != nil {
		return nil, err
	}
	return news, nil
}

// GetFundamentalsStatements retrieves fundamental statements for a ticker.
func (c *RestClient) GetFundamentalsStatements(ctx context.Context, ticker string, opts *FundamentalsStatementsOptions) ([]*FundamentalsStatement, error) {
	path := fmt.Sprintf("/tiingo/fundamentals/%s/statements", ticker)
	var statements []*FundamentalsStatement
	err := c.request(ctx, http.MethodGet, path, nil, opts, &statements)
	if err != nil {
		return nil, err
	}
	return statements, nil
}

// GetFundamentalsDaily retrieves daily fundamental data for a ticker.
func (c *RestClient) GetFundamentalsDaily(ctx context.Context, ticker string, opts *FundamentalsDailyOptions) ([]*FundamentalsStatement, error) {
	path := fmt.Sprintf("/tiingo/fundamentals/%s/daily", ticker)
	var fundamentals []*FundamentalsStatement
	err := c.request(ctx, http.MethodGet, path, nil, opts, &fundamentals)
	if err != nil {
		return nil, err
	}
	return fundamentals, nil
}

// Search retrieves search results for a query.
func (c *RestClient) Search(ctx context.Context, opts *SearchOptions) ([]*SearchResult, error) {
	path := "/tiingo/utilities/search"
	var results []*SearchResult
	err := c.request(ctx, http.MethodGet, path, nil, opts, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// handleHTTPError centralizes REST API error handling based on status codes.
func handleHTTPError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("bad request: invalid parameters (status %d)", resp.StatusCode)
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: invalid API key (status %d)", resp.StatusCode)
	case http.StatusNotFound:
		return fmt.Errorf("not found: ticker or endpoint does not exist (status %d)", resp.StatusCode)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limited: too many requests (status %d)", resp.StatusCode)
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return fmt.Errorf("server error (status %d)", resp.StatusCode)
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
