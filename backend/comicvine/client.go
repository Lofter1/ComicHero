package comicvine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL   = "https://comicvine.gamespot.com/api"
	defaultUserAgent = "comicvine-go-client/1.0"
	apiFormat        = "json"
)

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the API
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithRateLimit sets the rate limit in requests per second
func WithRateLimit(rps int) ClientOption {
	return func(c *Client) {
		c.rateLimiter = time.NewTicker(time.Second / time.Duration(rps))
	}
}

// WithRetry sets the maximum number of retry attempts
func WithRetry(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// WithUserAgent sets a custom User-Agent header
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// Client represents a Comic Vine API client
type Client struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	rateLimiter *time.Ticker
	maxRetries  int
	userAgent   string

	// Resource services
	characters *CharacterService
	concepts   *ConceptService
	episodes   *EpisodeService
	issues     *IssueService
	locations  *LocationService
	movies     *MovieService
	objects    *ObjectService
	origins    *OriginService
	powers     *PowerService
	publishers *PublisherService
	series     *SeriesService
	storyArcs  *StoryArcService
	teams      *TeamService
	volumes    *VolumeService
	promos     *PromoService
	videos     *VideoService
}

// NewClient creates a new Comic Vine API client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		maxRetries: 3,
		userAgent:  defaultUserAgent,
	}

	for _, opt := range opts {
		opt(c)
	}

	// Initialize services
	c.characters = &CharacterService{BaseService: newBaseService(c, ResourceCharacter)}
	c.concepts = &ConceptService{BaseService: newBaseService(c, ResourceConcept)}
	c.episodes = &EpisodeService{BaseService: newBaseService(c, ResourceEpisode)}
	c.issues = &IssueService{BaseService: newBaseService(c, ResourceIssue)}
	c.locations = &LocationService{BaseService: newBaseService(c, ResourceLocation)}
	c.movies = &MovieService{BaseService: newBaseService(c, ResourceMovie)}
	c.objects = &ObjectService{BaseService: newBaseService(c, ResourceObject)}
	c.origins = &OriginService{BaseService: newBaseService(c, ResourceOrigin)}
	c.powers = &PowerService{BaseService: newBaseService(c, ResourcePower)}
	c.publishers = &PublisherService{BaseService: newBaseService(c, ResourcePublisher)}
	c.series = &SeriesService{BaseService: newBaseService(c, ResourceSeries)}
	c.storyArcs = &StoryArcService{BaseService: newBaseService(c, ResourceStoryArc)}
	c.teams = &TeamService{BaseService: newBaseService(c, ResourceTeam)}
	c.volumes = &VolumeService{BaseService: newBaseService(c, ResourceVolume)}
	c.promos = &PromoService{BaseService: newBaseService(c, ResourcePromo)}
	c.videos = &VideoService{BaseService: newBaseService(c, ResourceVideo)}

	return c
}

// Resource accessors
func (c *Client) Characters() *CharacterService { return c.characters }
func (c *Client) Concepts() *ConceptService     { return c.concepts }
func (c *Client) Episodes() *EpisodeService     { return c.episodes }
func (c *Client) Issues() *IssueService         { return c.issues }
func (c *Client) Locations() *LocationService   { return c.locations }
func (c *Client) Movies() *MovieService         { return c.movies }
func (c *Client) Objects() *ObjectService       { return c.objects }
func (c *Client) Origins() *OriginService       { return c.origins }
func (c *Client) Powers() *PowerService         { return c.powers }
func (c *Client) Publishers() *PublisherService { return c.publishers }
func (c *Client) Series() *SeriesService        { return c.series }
func (c *Client) StoryArcs() *StoryArcService   { return c.storyArcs }
func (c *Client) Teams() *TeamService           { return c.teams }
func (c *Client) Volumes() *VolumeService       { return c.volumes }
func (c *Client) Promos() *PromoService         { return c.promos }
func (c *Client) Videos() *VideoService         { return c.videos }

// do performs an HTTP request with retries and rate limiting
func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if c.rateLimiter != nil {
		select {
		case <-c.rateLimiter.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err = c.httpClient.Do(req.WithContext(ctx))
		if err != nil {
			if attempt < c.maxRetries {
				time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, err)
		}

		if resp.StatusCode < 500 {
			break
		}

		resp.Body.Close()
		if attempt < c.maxRetries {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
	}

	return resp, nil
}

// get performs a GET request to the API
func (c *Client) get(ctx context.Context, path string, params url.Values) (*http.Response, error) {
	if params == nil {
		params = url.Values{}
	}

	params.Set("api_key", c.apiKey)
	params.Set("format", apiFormat)

	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, path, params.Encode())

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	return c.do(ctx, req)
}

// getJSON performs a GET request and decodes the JSON response
func (c *Client) getJSON(ctx context.Context, path string, params url.Values, v interface{}) error {
	resp, err := c.get(ctx, path, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return NewAPIError(resp.StatusCode, body)
	}

	// Check for API error response
	var apiError struct {
		Error  string `json:"error"`
		Status int    `json:"status_code"`
	}
	if err := json.Unmarshal(body, &apiError); err == nil && apiError.Status != 1 {
		return &APIError{
			StatusCode: apiError.Status,
			Message:    apiError.Error,
		}
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}
