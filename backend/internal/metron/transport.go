package metron

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *Client) get(ctx context.Context, path string, values url.Values, target any) error {
	_, err := c.getConditional(ctx, path, values, ConditionalRequest{}, target)
	return err
}

func (c *Client) getConditional(ctx context.Context, path string, values url.Values, conditional ConditionalRequest, target any) (FetchInfo, error) {
	if err := c.waitForRateLimit(ctx); err != nil {
		return FetchInfo{}, err
	}
	requestURL := c.requestURL(path)
	if len(values) > 0 {
		requestURL += "?" + values.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return FetchInfo{}, err
	}
	c.authorize(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ComicHero/0.1")
	conditionalHeader := false

	started := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordRequest(req, 0, started, conditionalHeader, err.Error())
		return FetchInfo{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	c.recordRequest(req, resp.StatusCode, started, conditionalHeader, "")

	rateLimit := c.updateRateLimit(resp.Header)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if resp.StatusCode == http.StatusTooManyRequests {
			return FetchInfo{}, &RateLimitError{
				Status:    resp.Status,
				Body:      strings.TrimSpace(string(body)),
				RateLimit: rateLimit,
			}
		}
		return FetchInfo{}, fmt.Errorf("metron request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchInfo{}, err
	}
	return FetchInfo{}, json.Unmarshal(body, target)
}

func (c *Client) recordRequest(req *http.Request, status int, started time.Time, conditional bool, errMessage string) {
	entry := RequestLogEntry{
		StartedAt:      started.UTC().Format(time.RFC3339),
		Method:         req.Method,
		URL:            req.URL.String(),
		Path:           req.URL.Path,
		Query:          req.URL.RawQuery,
		Status:         status,
		DurationMillis: time.Since(started).Milliseconds(),
		Conditional:    conditional,
		Error:          errMessage,
	}

	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.requestLog = append([]RequestLogEntry{entry}, c.requestLog...)
	if len(c.requestLog) > 200 {
		c.requestLog = c.requestLog[:200]
	}
}

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

func (c *Client) updateRateLimit(header http.Header) RateLimit {
	rateLimit := RateLimit{
		BurstLimit:         headerInt(header, "X-RateLimit-Burst-Limit"),
		BurstRemaining:     headerInt(header, "X-RateLimit-Burst-Remaining"),
		BurstReset:         headerInt64(header, "X-RateLimit-Burst-Reset"),
		SustainedLimit:     headerInt(header, "X-RateLimit-Sustained-Limit"),
		SustainedRemaining: headerInt(header, "X-RateLimit-Sustained-Remaining"),
		SustainedReset:     headerInt64(header, "X-RateLimit-Sustained-Reset"),
	}
	if rateLimit.Empty() {
		return c.CurrentRateLimit()
	}

	c.rateMu.Lock()
	defer c.rateMu.Unlock()
	c.rateLimit = rateLimit
	return rateLimit
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

func (c *Client) authorize(req *http.Request) {
	if c.username != "" || c.password != "" {
		credentials := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+credentials)
	}
}

func (c *Client) requestURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + path
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
