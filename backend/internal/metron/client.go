package metron

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	cache      map[string]cachedResponse
	cacheMu    sync.RWMutex
	rateLimit  RateLimit
	rateMu     sync.RWMutex
}

type Config struct {
	BaseURL  string
	Username string
	Password string
}

type Issue struct {
	ID          int    `json:"id"          doc:"Metron issue identifier." example:"123456"`
	Title       string `json:"title"       doc:"Issue title." example:"The Court of Owls"`
	Series      string `json:"series"      doc:"Series name." example:"Batman"`
	SeriesYear  int    `json:"seriesYear"  doc:"Series start year or volume year used by ComicHero generated titles." example:"2011"`
	Issue       int    `json:"issue"       doc:"Numeric issue number parsed for local sorting." example:"6"`
	Number      string `json:"number"      doc:"Original Metron issue number." example:"6"`
	Publisher   string `json:"publisher"   doc:"Publisher name." example:"DC Comics"`
	CoverDate   string `json:"coverDate"   doc:"Cover date as provided by Metron." example:"2012-04-01"`
	CoverImage  string `json:"coverImage"  doc:"Absolute Metron cover-image URL." format:"uri" example:"https://static.metron.cloud/media/issue/cover.jpg"`
	Description string `json:"description" doc:"Metron issue synopsis."`
}

type ReadingList struct {
	ID          int     `json:"id"          doc:"Metron reading-list identifier." example:"9876"`
	Name        string  `json:"name"        doc:"Metron reading-list name." example:"Batman: Court of Owls"`
	Description string  `json:"description" doc:"Metron reading-list description."`
	Issues      []Issue `json:"issues"      doc:"Issues included in the reading list, when requested from a detail endpoint."`
}

type Series struct {
	ID          int    `json:"id"                doc:"Metron series identifier." example:"405"`
	Name        string `json:"name"              doc:"Series name." example:"Batman"`
	Publisher   string `json:"publisher"         doc:"Publisher name." example:"DC Comics"`
	Volume      int    `json:"volume"            doc:"Series volume number." example:"2"`
	YearBegan   int    `json:"yearBegan"         doc:"First publication year." example:"2011"`
	YearEnd     int    `json:"yearEnd,omitempty" doc:"Final publication year, when the series has ended." example:"2016"`
	IssueCount  int    `json:"issueCount"        doc:"Number of issues reported by Metron." example:"52"`
	Description string `json:"description"       doc:"Metron series description."`
}

type RateLimit struct {
	BurstLimit         int   `json:"burstLimit,omitempty"         doc:"Metron burst-rate request limit."`
	BurstRemaining     int   `json:"burstRemaining,omitempty"     doc:"Remaining Metron burst-rate requests."`
	BurstReset         int64 `json:"burstReset,omitempty"         doc:"Unix timestamp when the burst-rate window resets."`
	SustainedLimit     int   `json:"sustainedLimit,omitempty"     doc:"Metron sustained-rate request limit."`
	SustainedRemaining int   `json:"sustainedRemaining,omitempty" doc:"Remaining Metron sustained-rate requests."`
	SustainedReset     int64 `json:"sustainedReset,omitempty"     doc:"Unix timestamp when the sustained-rate window resets."`
}

type RateLimitError struct {
	Status    string
	Body      string
	RateLimit RateLimit
}

func (e *RateLimitError) Error() string {
	reset := e.RateLimit.NextReset()
	if reset == 0 {
		return "metron rate limit reached"
	}
	return fmt.Sprintf("metron rate limit reached; try again after %s", time.Unix(reset, 0).Format(time.RFC3339))
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
		cache:    make(map[string]cachedResponse),
	}
}

func (c *Client) SearchIssues(ctx context.Context, query, series string, issue int) ([]Issue, error) {
	values := url.Values{}
	if series != "" {
		values.Set("series_name", series)
	} else if query != "" {
		values.Set("series_name", query)
	}
	if issue > 0 {
		values.Set("number", fmt.Sprintf("%d", issue))
	}

	results, err := c.getList(ctx, "/issue/", values)
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(results))
	for _, raw := range results {
		issues = append(issues, issueFromMap(raw))
	}
	return issues, nil
}

func (c *Client) GetIssue(ctx context.Context, id int) (*Issue, error) {
	var raw map[string]any
	if err := c.get(ctx, fmt.Sprintf("/issue/%d/", id), nil, &raw); err != nil {
		return nil, err
	}

	issue := issueFromMap(raw)
	return &issue, nil
}

func (c *Client) SearchReadingLists(ctx context.Context, query string) ([]ReadingList, error) {
	values := url.Values{}
	if query != "" {
		values.Set("name", query)
	}

	results, err := c.getList(ctx, "/reading_list/", values)
	if err != nil {
		return nil, err
	}
	lists := make([]ReadingList, 0, len(results))
	for _, raw := range results {
		lists = append(lists, readingListFromMap(raw))
	}
	return lists, nil
}

func (c *Client) SearchSeries(ctx context.Context, query string) ([]Series, error) {
	values := url.Values{}
	if query != "" {
		values.Set("name", query)
	}

	results, err := c.getList(ctx, "/series/", values)
	if err != nil {
		return nil, err
	}
	series := make([]Series, 0, len(results))
	for _, raw := range results {
		series = append(series, seriesFromMap(raw))
	}
	return series, nil
}

func (c *Client) GetSeriesIssues(ctx context.Context, id int) ([]Issue, error) {
	results, err := c.getAllList(ctx, fmt.Sprintf("/series/%d/issue_list/", id), nil)
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, 0, len(results))
	for _, raw := range results {
		issues = append(issues, issueFromMap(raw))
	}
	return issues, nil
}

func (c *Client) GetReadingList(ctx context.Context, id int) (*ReadingList, error) {
	var raw map[string]any
	if err := c.get(ctx, fmt.Sprintf("/reading_list/%d/", id), nil, &raw); err != nil {
		return nil, err
	}

	list := readingListFromMap(raw)
	issues, err := c.GetReadingListIssues(ctx, id)
	if err != nil {
		return nil, err
	}
	list.Issues = issues
	return &list, nil
}

func (c *Client) GetReadingListIssues(ctx context.Context, id int) ([]Issue, error) {
	results, err := c.getAllList(ctx, fmt.Sprintf("/reading_list/%d/items/", id), nil)
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, 0, len(results))
	for _, raw := range results {
		if issue := object(raw, "issue"); len(issue) > 0 {
			issues = append(issues, issueFromMap(issue))
		}
	}
	return issues, nil
}

func (c *Client) get(ctx context.Context, path string, values url.Values, target any) error {
	if err := c.waitForRateLimit(ctx); err != nil {
		return err
	}

	requestURL := c.requestURL(path)
	if len(values) > 0 {
		requestURL += "?" + values.Encode()
	}

	cacheKey := requestURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ComicHero/0.1")
	if cached, ok := c.cached(cacheKey); ok && cached.lastModified != "" {
		req.Header.Set("If-Modified-Since", cached.lastModified)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		c.updateRateLimit(resp.Header)
		if cached, ok := c.cached(cacheKey); ok {
			return json.Unmarshal(cached.body, target)
		}
	}

	rateLimit := c.updateRateLimit(resp.Header)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		if resp.StatusCode == http.StatusTooManyRequests {
			return &RateLimitError{
				Status:    resp.Status,
				Body:      strings.TrimSpace(string(body)),
				RateLimit: rateLimit,
			}
		}
		return fmt.Errorf("metron request failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	c.storeCache(cacheKey, resp.Header.Get("Last-Modified"), body)

	return json.Unmarshal(body, target)
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
		return listPage{results: page.Results, next: page.Next}, nil
	}

	var results []map[string]any
	if err := json.Unmarshal(raw, &results); err != nil {
		return listPage{}, err
	}
	return listPage{results: results}, nil
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

func (c *Client) cached(key string) (cachedResponse, bool) {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	cached, ok := c.cache[key]
	return cached, ok
}

func (c *Client) storeCache(key, lastModified string, body []byte) {
	if lastModified == "" {
		return
	}
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache[key] = cachedResponse{
		lastModified: lastModified,
		body:         append([]byte(nil), body...),
	}
}

type cachedResponse struct {
	lastModified string
	body         []byte
}

type pagedResponse struct {
	Results []map[string]any `json:"results"`
	Next    string           `json:"next"`
}

type listPage struct {
	results []map[string]any
	next    string
}

func issueFromMap(raw map[string]any) Issue {
	series := object(raw, "series")
	publisher := object(raw, "publisher")
	if len(publisher) == 0 {
		publisher = object(series, "publisher")
	}

	return Issue{
		ID:          intValue(raw, "id"),
		Title:       firstString(raw, "title", "name"),
		Series:      fallbackString(firstString(raw, "series_name", "series"), firstString(series, "name")),
		SeriesYear:  firstInt(intValue(raw, "series_year", "year_began", "year"), intValue(series, "year_began", "year")),
		Issue:       intValue(raw, "number", "issue"),
		Number:      firstString(raw, "number", "issue"),
		Publisher:   fallbackString(firstString(raw, "publisher_name", "publisher"), firstString(publisher, "name")),
		CoverDate:   firstString(raw, "cover_date", "store_date", "date"),
		CoverImage:  firstString(raw, "image", "cover", "cover_image", "image_url"),
		Description: firstString(raw, "desc", "description", "synopsis"),
	}
}

func firstInt(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func readingListFromMap(raw map[string]any) ReadingList {
	list := ReadingList{
		ID:          intValue(raw, "id"),
		Name:        firstString(raw, "name", "title"),
		Description: firstString(raw, "desc", "description", "summary"),
	}

	for _, key := range []string{"issues", "issue", "comics"} {
		values, ok := raw[key].([]any)
		if !ok {
			continue
		}
		for _, value := range values {
			if item, ok := value.(map[string]any); ok {
				list.Issues = append(list.Issues, issueFromMap(item))
			}
		}
	}
	return list
}

func seriesFromMap(raw map[string]any) Series {
	publisher := object(raw, "publisher")

	return Series{
		ID:          intValue(raw, "id"),
		Name:        firstString(raw, "name", "series"),
		Publisher:   fallbackString(firstString(raw, "publisher_name", "publisher"), firstString(publisher, "name")),
		Volume:      intValue(raw, "volume"),
		YearBegan:   intValue(raw, "year_began"),
		YearEnd:     intValue(raw, "year_end"),
		IssueCount:  intValue(raw, "issue_count"),
		Description: firstString(raw, "desc", "description"),
	}
}

func firstString(raw map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed
		case map[string]any:
			if name := firstString(typed, "url", "original_url", "image", "thumb_url", "name", "title"); name != "" {
				return name
			}
		}
	}
	return ""
}

func fallbackString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func object(raw map[string]any, key string) map[string]any {
	value, ok := raw[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func intValue(raw map[string]any, keys ...string) int {
	for _, key := range keys {
		switch value := raw[key].(type) {
		case float64:
			return int(value)
		case int:
			return value
		case string:
			var parsed int
			if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
				return parsed
			}
		}
	}
	return 0
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
