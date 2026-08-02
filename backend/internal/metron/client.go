package metron

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

const DefaultBaseURL = "https://metron.cloud/api"

type Client struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
	token      string
	rateLimit  RateLimit
	rateMu     sync.RWMutex
	requestMu  sync.RWMutex
	requestLog []RequestLogEntry
}

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

func (c *Client) CurrentRateLimit() RateLimit {
	c.rateMu.RLock()
	defer c.rateMu.RUnlock()
	return c.rateLimit
}

func (c *Client) RecentRequests() []RequestLogEntry {
	c.requestMu.RLock()
	defer c.requestMu.RUnlock()
	requests := make([]RequestLogEntry, len(c.requestLog))
	copy(requests, c.requestLog)
	return requests
}

func New(config Config) *Client {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		username: config.Username,
		password: config.Password,
		token:    config.Token,
	}
}
