package metron

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// PagedResponse is the envelope Metron wraps every list endpoint in.
type PagedResponse[T any] struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []T    `json:"results"`
}

// requestURL resolves a possibly-relative Metron path against the
// configured base URL. Absolute URLs (as returned in PagedResponse.Next)
// are passed through unchanged.
func (c *Client) requestURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + path
}

// get performs an authenticated GET request against path and decodes the
// JSON response body into v.
func (c *Client) get(ctx context.Context, path string, values url.Values, v any) error {
	requestURL := c.requestURL(path)
	if len(values) > 0 {
		requestURL += "?" + values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("metron: creating request: %w", err)
	}
	_, err = c.do(req, v)
	return err
}

// post performs an authenticated POST request with a JSON-encoded body and
// decodes the JSON response into v. v may be nil for endpoints that return
// no body (e.g. 204 No Content).
func (c *Client) post(ctx context.Context, path string, body any, v any) error {
	return c.send(ctx, http.MethodPost, path, body, v)
}

// patch performs an authenticated PATCH request with a JSON-encoded body.
func (c *Client) patch(ctx context.Context, path string, body any, v any) error {
	return c.send(ctx, http.MethodPatch, path, body, v)
}

// put performs an authenticated PUT request with a JSON-encoded body.
func (c *Client) put(ctx context.Context, path string, body any, v any) error {
	return c.send(ctx, http.MethodPut, path, body, v)
}

// delete performs an authenticated DELETE request. Metron's delete
// endpoints return no body.
func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.requestURL(path), nil)
	if err != nil {
		return fmt.Errorf("metron: creating request: %w", err)
	}
	_, err = c.do(req, nil)
	return err
}

func (c *Client) send(ctx context.Context, method, path string, body any, v any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("metron: encoding request body: %w", err)
		}
		reader = strings.NewReader(string(encoded))
	}

	req, err := http.NewRequestWithContext(ctx, method, c.requestURL(path), reader)
	if err != nil {
		return fmt.Errorf("metron: creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	_, err = c.do(req, v)
	return err
}

func (c *Client) do(req *http.Request, v any) (int, error) {
	c.authorize(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("metron: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	c.setRateLimit(rateLimitFromHeader(resp.Header))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("metron: reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return resp.StatusCode, &RateLimitError{RateLimit: c.CurrentRateLimit(), Body: strings.TrimSpace(string(body))}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, NewAPIError(resp.StatusCode, body)
	}

	if v == nil || len(body) == 0 {
		return resp.StatusCode, nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return resp.StatusCode, fmt.Errorf("metron: decoding response: %w", err)
	}
	return resp.StatusCode, nil
}

// Do performs an authenticated request against an arbitrary Metron API
// path using the given HTTP method and optional query values, JSON
// encoding body when non-nil and decoding the JSON response into target.
// It returns the HTTP status code alongside any error so callers that need
// it for logging or diagnostics don't have to parse it back out of the
// error.
//
// Do is the same building block the resource-specific service methods
// (Client.Arcs(), Client.Issues(), and so on) are implemented on top of.
// It is exported for callers - such as ComicHero's own
// backend/internal/metron domain-mapping layer - that need Metron's raw,
// loosely-typed JSON shapes (e.g. into a map[string]any or
// json.RawMessage) rather than this package's typed models, while still
// sharing this Client's authentication, base URL, and rate-limit tracking.
func (c *Client) Do(ctx context.Context, method, path string, values url.Values, body any, target any) (int, error) {
	requestURL := c.requestURL(path)
	if len(values) > 0 {
		requestURL += "?" + values.Encode()
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, fmt.Errorf("metron: encoding request body: %w", err)
		}
		reader = strings.NewReader(string(encoded))
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, reader)
	if err != nil {
		return 0, fmt.Errorf("metron: creating request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req, target)
}

func rateLimitFromHeader(header http.Header) RateLimit {
	return RateLimit{
		BurstLimit:         headerInt(header, "X-RateLimit-Burst-Limit"),
		BurstRemaining:     headerInt(header, "X-RateLimit-Burst-Remaining"),
		BurstReset:         headerInt64(header, "X-RateLimit-Burst-Reset"),
		SustainedLimit:     headerInt(header, "X-RateLimit-Sustained-Limit"),
		SustainedRemaining: headerInt(header, "X-RateLimit-Sustained-Remaining"),
		SustainedReset:     headerInt64(header, "X-RateLimit-Sustained-Reset"),
	}
}

func headerInt(header http.Header, key string) int {
	value, err := strconv.Atoi(header.Get(key))
	if err != nil {
		return 0
	}
	return value
}

func headerInt64(header http.Header, key string) int64 {
	value, err := strconv.ParseInt(header.Get(key), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

// listPage fetches a single page from a Metron list endpoint.
func listPage[T any](ctx context.Context, c *Client, path string, values url.Values) (PagedResponse[T], error) {
	var page PagedResponse[T]
	if err := c.get(ctx, path, values, &page); err != nil {
		return PagedResponse[T]{}, err
	}
	return page, nil
}

// listAll walks every page of a Metron list endpoint, starting at path with
// the given query values, and returns every result concatenated together.
// Metron includes an absolute "next" URL in each page, so subsequent
// requests reuse it directly and no longer need the original query values.
func listAll[T any](ctx context.Context, c *Client, path string, values url.Values) ([]T, error) {
	all := []T{}
	next := path
	for next != "" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var page PagedResponse[T]
		if err := c.get(ctx, next, values, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Results...)
		next = page.Next
		values = nil
	}
	return all, nil
}

// setPage sets the page query parameter when page is positive, matching
// Metron's ?page=N convention.
func setPage(values url.Values, page int) url.Values {
	if values == nil {
		values = url.Values{}
	}
	if page > 0 {
		values.Set("page", strconv.Itoa(page))
	}
	return values
}
