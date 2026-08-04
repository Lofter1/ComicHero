package metron

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the production Metron API root.
const DefaultBaseURL = "https://metron.cloud/api"

// defaultUserAgent identifies this client to Metron. Callers embedding this
// package in their own application should override it with WithUserAgent so
// Metron's operators can see which application is calling.
const defaultUserAgent = "metron-go-client/1.0"

// Client is a Metron API client. Create one with NewClient.
type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string

	username string
	password string
	cookie   string
	token    string

	rateMu    sync.RWMutex
	rateLimit RateLimit

	// Resource services. These are cheap value-holders around the shared
	// client and are safe to keep as fields rather than re-allocating on
	// every accessor call.
	arcs         *ArcService
	characters   *CharacterService
	collection   *CollectionService
	creators     *CreatorService
	imprints     *ImprintService
	issues       *IssueService
	publishers   *PublisherService
	pullLists    *PullListService
	readingLists *ReadingListService
	roles        *RoleService
	series       *SeriesService
	seriesTypes  *SeriesTypeService
	teams        *TeamService
	universes    *UniverseService
	wishLists    *WishListService
}

// ClientOption configures a Client returned by NewClient.
type ClientOption func(*Client)

// WithBaseURL overrides the default production API root. Useful for
// pointing at a staging environment or a test double.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		if trimmed := strings.TrimRight(baseURL, "/"); trimmed != "" {
			c.baseURL = trimmed
		}
	}
}

// WithHTTPClient overrides the default *http.Client, e.g. to set a custom
// transport or a shorter timeout.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		if userAgent != "" {
			c.userAgent = userAgent
		}
	}
}

// WithBasicAuth configures HTTP Basic Auth using a Metron username and
// password. Metron still accepts this scheme, but token auth
// (WithToken) is preferred where available since it is independently
// revocable and isn't the account password.
func WithBasicAuth(username, password string) ClientOption {
	return func(c *Client) {
		c.username = username
		c.password = password
	}
}

// WithCookie configures session-cookie authentication using the value of
// Metron's sessionid cookie.
func WithCookie(sessionID string) ClientOption {
	return func(c *Client) {
		c.cookie = sessionID
	}
}

// WithToken configures bearer-token authentication using a Metron API
// token. It takes priority over Basic Auth and cookie auth when set.
func WithToken(token string) ClientOption {
	return func(c *Client) {
		c.token = token
	}
}

// NewClient creates a Metron API client. Without any options the client is
// unauthenticated, which only permits anonymous-readable endpoints (Metron
// requires authentication for most resources).
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 20 * time.Second},
		userAgent:  defaultUserAgent,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.arcs = &ArcService{client: c}
	c.characters = &CharacterService{client: c}
	c.collection = &CollectionService{client: c}
	c.creators = &CreatorService{client: c}
	c.imprints = &ImprintService{client: c}
	c.issues = &IssueService{client: c}
	c.publishers = &PublisherService{client: c}
	c.pullLists = &PullListService{client: c}
	c.readingLists = &ReadingListService{client: c}
	c.roles = &RoleService{client: c}
	c.series = &SeriesService{client: c}
	c.seriesTypes = &SeriesTypeService{client: c}
	c.teams = &TeamService{client: c}
	c.universes = &UniverseService{client: c}
	c.wishLists = &WishListService{client: c}

	return c
}

// Resource accessors, one per Metron API tag/resource group.
func (c *Client) Arcs() *ArcService                 { return c.arcs }
func (c *Client) Characters() *CharacterService     { return c.characters }
func (c *Client) Collection() *CollectionService    { return c.collection }
func (c *Client) Creators() *CreatorService         { return c.creators }
func (c *Client) Imprints() *ImprintService         { return c.imprints }
func (c *Client) Issues() *IssueService             { return c.issues }
func (c *Client) Publishers() *PublisherService     { return c.publishers }
func (c *Client) PullLists() *PullListService       { return c.pullLists }
func (c *Client) ReadingLists() *ReadingListService { return c.readingLists }
func (c *Client) Roles() *RoleService               { return c.roles }
func (c *Client) Series() *SeriesService            { return c.series }
func (c *Client) SeriesTypes() *SeriesTypeService   { return c.seriesTypes }
func (c *Client) Teams() *TeamService               { return c.teams }
func (c *Client) Universes() *UniverseService       { return c.universes }
func (c *Client) WishLists() *WishListService       { return c.wishLists }

// CurrentRateLimit returns the most recently observed rate-limit state, as
// reported by Metron's X-RateLimit-* response headers. The zero value means
// no request has completed yet.
func (c *Client) CurrentRateLimit() RateLimit {
	c.rateMu.RLock()
	defer c.rateMu.RUnlock()
	return c.rateLimit
}

func (c *Client) setRateLimit(rl RateLimit) {
	if rl.Empty() {
		return
	}
	c.rateMu.Lock()
	defer c.rateMu.Unlock()
	c.rateLimit = rl
}

// authorize applies whichever Metron credential is configured to an
// outgoing request. A token takes priority when configured, then a cookie,
// then Basic Auth.
func (c *Client) authorize(req *http.Request) {
	switch {
	case c.token != "":
		req.Header.Set("Authorization", "Bearer "+c.token)
	case c.cookie != "":
		req.AddCookie(&http.Cookie{Name: "sessionid", Value: c.cookie})
	case c.username != "" || c.password != "":
		req.SetBasicAuth(c.username, c.password)
	}
}
