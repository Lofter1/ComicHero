package comicvine

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError represents an error returned by the Comic Vine API
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"error"`
	Body       string `json:"-"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("Comic Vine API error (status %d): %s", e.StatusCode, e.Message)
}

// NewAPIError creates a new APIError from an HTTP response
func NewAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Body:       string(body),
	}

	// Try to parse the error message from the response body
	var errResp struct {
		Error  string `json:"error"`
		Status int    `json:"status_code"`
	}

	if err := json.Unmarshal(body, &errResp); err == nil {
		apiErr.Message = errResp.Error
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(statusCode)
	}

	return apiErr
}

// IsNotFound returns true if the error represents a 404 Not Found
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error represents a 401 Unauthorized
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsRateLimited returns true if the error represents a 429 Too Many Requests
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// IsServerError returns true if the error represents a 5xx server error
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsClientError returns true if the error represents a 4xx client error
func (e *APIError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}
