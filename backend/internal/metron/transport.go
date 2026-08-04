package metron

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	metronapi "github.com/Lofter1/ComicHero/backend/metron"
)

func (c *Client) get(ctx context.Context, path string, values url.Values, target any) error {
	_, err := c.getConditional(ctx, path, values, target)
	return err
}

// getConditional performs a GET request against path. It no longer sends
// conditional-request headers (Metron doesn't return partial results for
// them the way ComicHero originally hoped), but keeps returning FetchInfo
// for source compatibility with existing callers.
func (c *Client) getConditional(ctx context.Context, path string, values url.Values, target any) (FetchInfo, error) {
	if err := c.waitForRateLimit(ctx); err != nil {
		return FetchInfo{}, err
	}

	started := time.Now()
	status, err := c.raw.Do(ctx, http.MethodGet, path, values, nil, target)
	c.recordRequest(path, values, status, started, err)

	if err != nil {
		var rateLimitErr *metronapi.RateLimitError
		if errors.As(err, &rateLimitErr) {
			return FetchInfo{}, &RateLimitError{
				Status:    fmt.Sprintf("%d %s", http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests)),
				Body:      rateLimitErr.Body,
				RateLimit: RateLimit(rateLimitErr.RateLimit),
			}
		}
		return FetchInfo{}, err
	}
	return FetchInfo{}, nil
}

func (c *Client) recordRequest(path string, values url.Values, status int, started time.Time, err error) {
	requestURL := path
	if len(values) > 0 {
		requestURL += "?" + values.Encode()
	}
	errMessage := ""
	if err != nil {
		errMessage = err.Error()
	}

	entry := RequestLogEntry{
		StartedAt:      started.UTC().Format(time.RFC3339),
		Method:         http.MethodGet,
		URL:            requestURL,
		Path:           path,
		Query:          values.Encode(),
		Status:         status,
		DurationMillis: time.Since(started).Milliseconds(),
		Error:          errMessage,
	}

	if parsed, parseErr := url.Parse(requestURL); parseErr == nil {
		entry.URL = parsed.String()
		entry.Path = parsed.Path
		entry.Query = parsed.RawQuery
	}

	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.requestLog = append([]RequestLogEntry{entry}, c.requestLog...)
	if len(c.requestLog) > 200 {
		c.requestLog = c.requestLog[:200]
	}
}

// waitForRateLimit proactively backs off ahead of a request when the
// previous response indicated Metron's rate-limit window is exhausted,
// rather than firing the request and handling the 429 reactively.
func (c *Client) waitForRateLimit(ctx context.Context) error {
	rateLimit := c.CurrentRateLimit()
	reset := rateLimit.NextReset()
	if reset == 0 {
		return nil
	}

	waitUntil := time.Unix(reset, 0).Add(time.Second)
	wait := time.Until(waitUntil)
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) getList(ctx context.Context, path string, values url.Values) ([]map[string]any, error) {
	var raw json.RawMessage
	if err := c.get(ctx, path, values, &raw); err != nil {
		return nil, err
	}

	var page pagedResponse
	if err := json.Unmarshal(raw, &page); err == nil && page.Results != nil {
		return page.Results, nil
	}

	var results []map[string]any
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (c *Client) getAllList(ctx context.Context, path string, values url.Values) ([]map[string]any, error) {
	var all []map[string]any
	next := path
	for next != "" {
		page, err := c.getListPage(ctx, next, values)
		if err != nil {
			return nil, err
		}
		all = append(all, page.results...)
		next = page.next
		values = nil
	}
	return all, nil
}

func (c *Client) getListPage(ctx context.Context, path string, values url.Values) (listPage, error) {
	var raw json.RawMessage
	if err := c.get(ctx, path, values, &raw); err != nil {
		return listPage{}, err
	}

	var page pagedResponse
	if err := json.Unmarshal(raw, &page); err == nil && page.Results != nil {
		return listPage{results: page.Results, next: page.Next, count: page.Count}, nil
	}

	var results []map[string]any
	if err := json.Unmarshal(raw, &results); err != nil {
		return listPage{}, err
	}
	return listPage{results: results, count: len(results)}, nil
}

type pagedResponse struct {
	Results []map[string]any `json:"results"`
	Next    string           `json:"next"`
	Count   int              `json:"count"`
}

type listPage struct {
	results []map[string]any
	next    string
	count   int
}
