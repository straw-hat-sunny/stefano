package websearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const defaultTavilyBaseURL = "https://api.tavily.com/"

// TavilyParams is the JSON body for POST /search. Authenticate with Bearer token on the request (see Client.Search).
type TavilyParams struct {
	Query           string `json:"query"`
	AutoParameters  bool   `json:"auto_parameters,omitempty"`
	Topic           string `json:"topic,omitempty"`
	SearchDepth     string `json:"search_depth,omitempty"`
	ChunksPerSource int    `json:"chunks_per_source,omitempty"`
	MaxResults      int    `json:"max_results,omitempty"`
	TimeRange       string `json:"time_range,omitempty"`
	StartDate       string `json:"start_date,omitempty"`
	EndDate         string `json:"end_date,omitempty"`
	IncludeAnswer   bool   `json:"include_answer,omitempty"`
	// No omitempty: a boxed false must appear in JSON; omitempty would drop it for any.
	IncludeRawContent        any      `json:"include_raw_content"` // bool or "markdown" | "text"
	IncludeImages            bool     `json:"include_images,omitempty"`
	IncludeImageDescriptions bool     `json:"include_image_descriptions,omitempty"`
	IncludeFavicon           bool     `json:"include_favicon,omitempty"`
	IncludeDomains           []string `json:"include_domains,omitempty"`
	ExcludeDomains           []string `json:"exclude_domains,omitempty"`
	Country                  string   `json:"country,omitempty"`
	IncludeUsage             bool     `json:"include_usage,omitempty"`
}

type TavilyParamOptions func(*TavilyParams)

func WithAutoParameters(autoParameters bool) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.AutoParameters = autoParameters
	}
}

func WithTopic(topic string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.Topic = topic
	}
}

func WithSearchDepth(searchDepth string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.SearchDepth = searchDepth
	}
}

func WithChunksPerSource(chunksPerSource int) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.ChunksPerSource = chunksPerSource
	}
}

func WithMaxResults(maxResults int) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.MaxResults = maxResults
	}
}

func WithTimeRange(timeRange string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.TimeRange = timeRange
	}
}

func WithStartDate(startDate string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.StartDate = startDate
	}
}

func WithEndDate(endDate string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.EndDate = endDate
	}
}

func WithIncludeAnswer(includeAnswer bool) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeAnswer = includeAnswer
	}
}

// WithIncludeRawContent sets include_raw_content to a boolean (API accepts true/false).
func WithIncludeRawContent(includeRawContent bool) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeRawContent = includeRawContent
	}
}

// WithIncludeRawContentFormat sets include_raw_content to "markdown" or "text" per Tavily Search API.
func WithIncludeRawContentFormat(format string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeRawContent = format
	}
}

func WithIncludeImages(includeImages bool) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeImages = includeImages
	}
}

func WithIncludeImageDescriptions(includeImageDescriptions bool) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeImageDescriptions = includeImageDescriptions
	}
}

func WithIncludeFavicon(includeFavicon bool) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeFavicon = includeFavicon
	}
}

func WithIncludeDomains(includeDomains []string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeDomains = includeDomains
	}
}

func WithExcludeDomains(excludeDomains []string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.ExcludeDomains = excludeDomains
	}
}

func WithCountry(country string) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.Country = country
	}
}

func WithIncludeUsage(includeUsage bool) TavilyParamOptions {
	return func(params *TavilyParams) {
		params.IncludeUsage = includeUsage
	}
}

// Client calls the Tavily Search API with credit-friendly defaults: search_depth=basic, max_results=5,
// include_raw_content="markdown".
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

type ClientOption func(*Client)

func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
		baseURL:    defaultTavilyBaseURL,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func defaultTavilyParams(query string) *TavilyParams {
	return &TavilyParams{
		Query:             query,
		SearchDepth:       "basic",
		MaxResults:        5,
		IncludeRawContent: false,
		IncludeAnswer:     true,
	}
}

// Search runs POST /search. Defaults preserve API credits (basic depth) and cap results at 5; raw page content is requested as markdown.
func (c *Client) Search(ctx context.Context, query string, opts ...TavilyParamOptions) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("websearch: empty query")
	}
	p := defaultTavilyParams(query)
	for _, o := range opts {
		o(p)
	}

	body, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("websearch: encode request: %w", err)
	}
	urlEndpoint, err := url.JoinPath(c.baseURL, "search")
	if err != nil {
		return nil, fmt.Errorf("websearch: join path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var out SearchResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("websearch: decode response: %w", err)
	}
	return &out, nil
}

type SearchResponse struct {
	Query          string          `json:"query"`
	Answer         string          `json:"answer"`
	Images         []ImageHit      `json:"images"`
	Results        []SearchResult  `json:"results"`
	ResponseTime   float64         `json:"response_time"`
	Usage          *UsageInfo      `json:"usage,omitempty"`
	RequestID      string          `json:"request_id,omitempty"`
	AutoParameters json.RawMessage `json:"auto_parameters,omitempty"`
}

type SearchResult struct {
	Title      string     `json:"title"`
	URL        string     `json:"url"`
	Content    string     `json:"content"`
	Score      float64    `json:"score"`
	RawContent string     `json:"raw_content"`
	Favicon    string     `json:"favicon,omitempty"`
	Images     []ImageHit `json:"images,omitempty"`
}

type ImageHit struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type UsageInfo struct {
	Credits int `json:"credits"`
}

type apiErrorBody struct {
	Detail struct {
		Error string `json:"error"`
	} `json:"detail"`
}

func parseAPIError(status int, body []byte) error {
	var wrapped apiErrorBody
	fmt.Printf("body: %s\n status: %d\n", string(body), status)
	msg := strings.TrimSpace(string(body))
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Detail.Error != "" {
		msg = wrapped.Detail.Error
	}
	return fmt.Errorf("websearch: tavily HTTP %d: %s", status, msg)
}
