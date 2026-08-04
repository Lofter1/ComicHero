package metron

import (
	"strings"
	"sync"

	metronapi "github.com/Lofter1/ComicHero/backend/metron"
)

// DefaultBaseURL is re-exported from backend/metron so existing callers
// that referenced this package's own constant keep compiling.
const DefaultBaseURL = metronapi.DefaultBaseURL

// userAgent identifies ComicHero to Metron.
const userAgent = "ComicHero/0.1"

// Client is ComicHero's Metron client. It wraps the general-purpose
// backend/metron SDK - which handles authentication, the base URL, and raw
// HTTP transport - with the application-specific behavior ComicHero needs
// on top: proactive rate-limit backoff before issuing a request, a rolling
// diagnostics log of recent requests, and decoding into the loosely-typed
// map[string]any shapes that mapping.go's Metron-to-ComicHero conversions
// are built around.
type Client struct {
	raw *metronapi.Client

	requestMu  sync.RWMutex
	requestLog []RequestLogEntry
}

// Config configures a Client returned by New.
type Config struct {
	BaseURL  string
	Username string
	Password string
	// Token is a Metron API token (see
	// https://metron-project.github.io/blog/token-authentication), sent as
	// `Authorization: Bearer <token>`. It takes priority over
	// Username/Password when set - Basic Auth remains supported as a
	// fallback since Metron has not removed it.
	Token string
}

func (r RateLimit) NextReset() int64 {
	var reset int64
	if r.BurstRemaining == 0 && r.BurstReset > reset {
		reset = r.BurstReset
	}
	if r.SustainedRemaining == 0 && r.SustainedReset > reset {
		reset = r.SustainedReset
	}
	return reset
}

func (r RateLimit) Empty() bool {
	return r == RateLimit{}
}

// CurrentRateLimit returns the most recently observed Metron rate-limit
// state.
func (c *Client) CurrentRateLimit() RateLimit {
	return RateLimit(c.raw.CurrentRateLimit())
}

// RecentRequests returns a snapshot of the most recent Metron requests
// this client has made, most-recent first, for diagnostics.
func (c *Client) RecentRequests() []RequestLogEntry {
	c.requestMu.RLock()
	defer c.requestMu.RUnlock()
	requests := make([]RequestLogEntry, len(c.requestLog))
	copy(requests, c.requestLog)
	return requests
}

// New creates a Metron client for the given configuration.
func New(config Config) *Client {
	opts := []metronapi.ClientOption{metronapi.WithUserAgent(userAgent)}
	if baseURL := strings.TrimRight(config.BaseURL, "/"); baseURL != "" {
		opts = append(opts, metronapi.WithBaseURL(baseURL))
	}
	if config.Token != "" {
		opts = append(opts, metronapi.WithToken(config.Token))
	} else if config.Username != "" || config.Password != "" {
		opts = append(opts, metronapi.WithBasicAuth(config.Username, config.Password))
	}

	return &Client{raw: metronapi.NewClient(opts...)}
}
