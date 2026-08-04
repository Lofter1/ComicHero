package metron

import (
	"fmt"
	"net/http"
	"time"
)

// APIError represents a non-2xx response from the Metron API.
type APIError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Body       string `json:"-"`
}

// NewAPIError builds an APIError from an HTTP status code and response
// body, using the body as the message when it looks like plain text and
// falling back to the standard HTTP status text otherwise.
func NewAPIError(statusCode int, body []byte) *APIError {
	message := string(body)
	if len(message) > 500 {
		message = message[:500]
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return &APIError{StatusCode: statusCode, Message: message, Body: string(body)}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("metron API error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound reports whether the error represents a 404 Not Found.
func (e *APIError) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }

// IsUnauthorized reports whether the error represents a 401 Unauthorized.
func (e *APIError) IsUnauthorized() bool { return e.StatusCode == http.StatusUnauthorized }

// IsForbidden reports whether the error represents a 403 Forbidden.
func (e *APIError) IsForbidden() bool { return e.StatusCode == http.StatusForbidden }

// IsServerError reports whether the error represents a 5xx server error.
func (e *APIError) IsServerError() bool { return e.StatusCode >= 500 }

// RateLimit describes Metron's burst and sustained request-rate windows, as
// reported by its X-RateLimit-* response headers.
type RateLimit struct {
	BurstLimit         int   `json:"burstLimit,omitempty"`
	BurstRemaining     int   `json:"burstRemaining,omitempty"`
	BurstReset         int64 `json:"burstReset,omitempty"`
	SustainedLimit     int   `json:"sustainedLimit,omitempty"`
	SustainedRemaining int   `json:"sustainedRemaining,omitempty"`
	SustainedReset     int64 `json:"sustainedReset,omitempty"`
}

// Empty reports whether no rate-limit headers were present.
func (r RateLimit) Empty() bool { return r == RateLimit{} }

// NextReset returns the unix timestamp at which the soonest-exhausted
// window resets, or 0 if neither window is exhausted.
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

// RateLimitError is returned when Metron responds 429 Too Many Requests.
type RateLimitError struct {
	RateLimit RateLimit
	Body      string
}

func (e *RateLimitError) Error() string {
	reset := e.RateLimit.NextReset()
	if reset == 0 {
		return "metron: rate limit reached"
	}
	return fmt.Sprintf("metron: rate limit reached; resets at %s", time.Unix(reset, 0).UTC().Format(time.RFC3339))
}
