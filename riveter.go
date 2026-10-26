// Package riveter is the official Go SDK for the Riveter API — enrich data,
// build datasets, scrape pages, and run web searches.
// https://docs.riveterhq.com
package riveter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.riveterhq.com/v1"

// Version is the SDK version reported in the User-Agent header.
const Version = "1.0.0"

// Client talks to the Riveter API. Create one with NewClient.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int

	Runs               *RunsService
	Enrichments        *EnrichmentsService
	Datasets           *DatasetsService
	ConfiguredDatasets *ConfiguredDatasetsService
	Extractions        *ExtractionsService
	Monitors           *MonitorsService
	Tools              *ToolsService
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the API key (default: the RIVETER_API_KEY environment variable).
func WithAPIKey(apiKey string) Option {
	return func(c *Client) { c.apiKey = apiKey }
}

// WithBaseURL sets the API base URL (default: RIVETER_BASE_URL, then production).
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets a custom *http.Client. Its Timeout must exceed the 50s
// `wait` long-poll (the default client uses 60s).
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithMaxRetries sets the extra attempts after a 429 (any method) or a
// 5xx/network failure (GETs only). Default 2.
func WithMaxRetries(maxRetries int) Option {
	return func(c *Client) { c.maxRetries = maxRetries }
}

// NewClient builds a Client, reading RIVETER_API_KEY and RIVETER_BASE_URL from
// the environment when not set through options.
func NewClient(opts ...Option) (*Client, error) {
	client := &Client{
		apiKey:     os.Getenv("RIVETER_API_KEY"),
		baseURL:    defaultBaseURL,
		maxRetries: 2,
	}
	if fromEnv := os.Getenv("RIVETER_BASE_URL"); fromEnv != "" {
		client.baseURL = fromEnv
	}
	for _, opt := range opts {
		opt(client)
	}
	if client.apiKey == "" {
		return nil, errors.New("riveter: missing API key: use WithAPIKey or set RIVETER_API_KEY (get one at https://app.riveterhq.com/settings/api)")
	}
	client.baseURL = strings.TrimRight(client.baseURL, "/")
	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	client.Runs = &RunsService{client: client}
	client.Enrichments = &EnrichmentsService{client: client}
	client.Datasets = &DatasetsService{client: client}
	client.ConfiguredDatasets = &ConfiguredDatasetsService{client: client}
	client.Extractions = &ExtractionsService{client: client}
	client.Monitors = &MonitorsService{client: client}
	client.Tools = &ToolsService{client: client}
	return client, nil
}

// Enrich starts an enrichment run on rows of input data. Rows come from Input
// (inline) or DatasetID (a Dataset); column config from
// EnrichmentID, Output, or Prompt + Attributes. On a Dataset, Columns / Rows /
// Positions (or Targets) scope the fill and States names the cell states to
// re-run (Force runs every cell). (operationId: enrich)
func (c *Client) Enrich(ctx context.Context, params EnrichParams) (*Run, error) {
	run := &Run{}
	if err := c.request(ctx, http.MethodPost, "/enrich", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
}

// QuickSearch runs one web search synchronously; results are already in
// Output. (operationId: quickSearch)
func (c *Client) QuickSearch(ctx context.Context, params QuickSearchParams) (*Run, error) {
	run := &Run{}
	if err := c.request(ctx, http.MethodPost, "/quick_search", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
}

// SearchAgent asks one research question and long-polls up to params.Wait
// seconds (default 50) for the answer in Output. (operationId: searchAgent)
func (c *Client) SearchAgent(ctx context.Context, params SearchAgentParams) (*Run, error) {
	run := &Run{}
	if err := c.request(ctx, http.MethodPost, "/search_agent", nil, params, run); err != nil {
		return nil, err
	}
	return run, nil
}

// Scrape scrapes a webpage and returns its text synchronously. (operationId: scrape)
func (c *Client) Scrape(ctx context.Context, params ScrapeParams) (*ScrapeResult, error) {
	result := &ScrapeResult{}
	if err := c.request(ctx, http.MethodPost, "/scrape", nil, params, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Account returns account, plan, and credit info for the API key. (operationId: getAccount)
func (c *Client) Account(ctx context.Context) (*AccountInfo, error) {
	info := &AccountInfo{}
	if err := c.request(ctx, http.MethodGet, "/account", nil, nil, info); err != nil {
		return nil, err
	}
	return info, nil
}

// requestBody is a request body sent as-is under its own content type (CSV imports).
type requestBody struct {
	content     []byte
	contentType string
}

// request JSON-encodes body (or sends a *requestBody as-is) and JSON-decodes the response
// into out. Pass a *[]byte as out to keep the raw body (exports); a 204 or empty body
// leaves out untouched.
func (c *Client) request(ctx context.Context, method, path string, query url.Values, body, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	encoded, err := encodeRequestBody(body)
	if err != nil {
		return err
	}

	retryOnFailure := method == http.MethodGet
	for attempt := 0; ; attempt++ {
		var reader io.Reader
		if encoded != nil {
			reader = bytes.NewReader(encoded.content)
		}
		request, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return fmt.Errorf("riveter: build request: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
		request.Header.Set("User-Agent", "riveter-sdk-go/"+Version)
		if encoded != nil {
			request.Header.Set("Content-Type", encoded.contentType)
		}

		response, err := c.httpClient.Do(request)
		if err != nil {
			if retryOnFailure && attempt < c.maxRetries && ctx.Err() == nil {
				sleepContext(ctx, backoffDelay(attempt))
				continue
			}
			return &ConnectionError{Err: err}
		}

		responseBody, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			return &ConnectionError{Err: readErr}
		}

		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return decodeSuccessBody(responseBody, out)
		}

		if response.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			sleepContext(ctx, rateLimitDelay(response, attempt))
			continue
		}
		if response.StatusCode >= 500 && retryOnFailure && attempt < c.maxRetries {
			sleepContext(ctx, backoffDelay(attempt))
			continue
		}

		return newAPIError(response.StatusCode, responseBody)
	}
}

func encodeRequestBody(body any) (*requestBody, error) {
	switch typed := body.(type) {
	case nil:
		return nil, nil
	case *requestBody:
		return typed, nil
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("riveter: encode request body: %w", err)
	}
	return &requestBody{content: payload, contentType: "application/json"}, nil
}

func decodeSuccessBody(responseBody []byte, out any) error {
	if out == nil || len(responseBody) == 0 {
		return nil
	}
	if raw, keepRaw := out.(*[]byte); keepRaw {
		*raw = responseBody
		return nil
	}
	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("riveter: decode response: %w", err)
	}
	return nil
}

func backoffDelay(attempt int) time.Duration {
	delay := 500 * time.Millisecond << attempt
	if delay > 8*time.Second {
		delay = 8 * time.Second
	}
	return delay
}

func rateLimitDelay(response *http.Response, attempt int) time.Duration {
	reset, err := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64)
	if err != nil || reset <= 0 {
		return backoffDelay(attempt)
	}
	delay := time.Until(time.Unix(reset, 0))
	if delay < 0 {
		delay = 0
	}
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	return delay
}

func sleepContext(ctx context.Context, delay time.Duration) {
	if delay <= 0 {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
