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
	baseURL     string
	httpClient  *http.Client
	username    string
	password    string
	cache       map[string]cachedResponse
	cacheMu     sync.RWMutex
	rateLimit   RateLimit
	rateMu      sync.RWMutex
	requestMu   sync.RWMutex
	requestLog  []RequestLogEntry
	intervalMu  sync.Mutex
	minInterval time.Duration
	nextRequest time.Time
}

// SetMinInterval sets the minimum time between starts of outbound Metron HTTP
// requests. It is applied in the client immediately, so scheduled and manual
// Metron operations share the same pacing boundary.
func (c *Client) SetMinInterval(interval time.Duration) {
	if interval < 0 {
		interval = 0
	}
	c.intervalMu.Lock()
	c.minInterval = interval
	if interval == 0 {
		c.nextRequest = time.Time{}
	}
	c.intervalMu.Unlock()
}

type Config struct {
	BaseURL  string
	Username string
	Password string
}

type Issue struct {
	ID           int               `json:"id"          doc:"Metron issue identifier." example:"123456"`
	Title        string            `json:"title"       doc:"Issue title." example:"The Court of Owls"`
	StoryNames   []string          `json:"storyNames"  doc:"Story titles returned by Metron."`
	SeriesID     int               `json:"seriesId"    doc:"Metron series identifier for this issue." example:"405"`
	Series       string            `json:"series"      doc:"Series name." example:"Batman"`
	SeriesYear   int               `json:"seriesYear"  doc:"Series start year or volume year used by ComicHero generated titles." example:"2011"`
	SeriesVolume int               `json:"seriesVolume" doc:"Metron series volume number." example:"2"`
	Issue        string            `json:"issue"       doc:"Issue number as provided by Metron." example:"6.LR"`
	Number       string            `json:"number"      doc:"Original Metron issue number." example:"6"`
	Publisher    string            `json:"publisher"   doc:"Publisher name." example:"DC Comics"`
	CoverDate    string            `json:"coverDate"   doc:"Cover date as provided by Metron." example:"2012-04-01"`
	StoreDate    string            `json:"storeDate"   doc:"Store date as provided by Metron." example:"2012-04-01"`
	CoverImage   string            `json:"coverImage"  doc:"Absolute Metron cover-image URL." format:"uri" example:"https://static.metron.cloud/media/issue/cover.jpg"`
	Description  string            `json:"description" doc:"Metron issue synopsis."`
	Modified     string            `json:"modified"    doc:"Metron modified timestamp."`
	Tags         []string          `json:"tags"        doc:"Reading-list item tags/classifications from Metron."`
	Arcs         []MetronArc       `json:"arcs"        doc:"Story arcs attached to the issue, when returned by Metron."`
	Characters   []MetronCharacter `json:"characters"  doc:"Characters appearing in the issue, when returned by Metron."`
}

type MetronCharacter struct {
	ID          int      `json:"id"          doc:"Metron character identifier." example:"100"`
	Name        string   `json:"name"        doc:"Character name." example:"Batman"`
	Aliases     []string `json:"aliases"     doc:"Known character aliases from Metron."`
	Description string   `json:"description" doc:"Character description from Metron."`
	Image       string   `json:"image"       doc:"Character image URL from Metron." format:"uri"`
}

type ReadingList struct {
	ID                int        `json:"id"                doc:"Metron reading-list identifier." example:"9876"`
	Name              string     `json:"name"              doc:"Metron reading-list name." example:"Batman: Court of Owls"`
	Slug              string     `json:"slug,omitempty"    doc:"Metron reading-list slug."`
	User              MetronUser `json:"user,omitempty"    doc:"Metron user who owns the reading list."`
	ListType          string     `json:"listType"          doc:"Metron reading-list type."`
	IsPrivate         bool       `json:"isPrivate"         doc:"Whether the Metron reading list is private."`
	AttributionSource string     `json:"attributionSource" doc:"Reading-list attribution source."`
	AttributionURL    string     `json:"attributionUrl"    doc:"Reading-list attribution URL." format:"uri"`
	AverageRating     float64    `json:"averageRating"     doc:"Average Metron user rating."`
	RatingCount       int        `json:"ratingCount"       doc:"Number of Metron ratings."`
	Modified          string     `json:"modified"          doc:"Metron modified timestamp."`
	Image             string     `json:"image"             doc:"Metron reading-list image URL." format:"uri"`
	ItemsURL          string     `json:"itemsUrl"          doc:"Metron reading-list items URL." format:"uri"`
	ResourceURL       string     `json:"resourceUrl"       doc:"Metron reading-list resource URL." format:"uri"`
	Description       string     `json:"description"       doc:"Metron reading-list description."`
	Issues            []Issue    `json:"issues"            doc:"Issues included in the reading list, when requested from a detail endpoint."`
}

type MetronUser struct {
	ID       int    `json:"id"       doc:"Metron user identifier." example:"42"`
	Username string `json:"username" doc:"Metron username." example:"reader"`
}

type MetronArc struct {
	ID          int     `json:"id"          doc:"Metron story-arc identifier." example:"9876"`
	Name        string  `json:"name"        doc:"Metron story-arc name." example:"Batman: Zero Year"`
	Description string  `json:"description" doc:"Metron story-arc description."`
	Image       string  `json:"image"       doc:"Metron story-arc image URL." format:"uri"`
	Modified    string  `json:"modified"    doc:"Metron modified timestamp."`
	Issues      []Issue `json:"issues"      doc:"Issues included in the story arc, when requested from a detail endpoint."`
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

type ConditionalRequest struct {
	LastModified string
	Force        bool
}

type FetchInfo struct {
	LastModified string
	NotModified  bool
}

type RequestLogEntry struct {
	StartedAt      string `json:"startedAt"`
	Method         string `json:"method"`
	URL            string `json:"url"`
	Path           string `json:"path"`
	Query          string `json:"query"`
	Status         int    `json:"status"`
	DurationMillis int64  `json:"durationMillis"`
	Conditional    bool   `json:"conditional"`
	Error          string `json:"error,omitempty"`
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
		cache:    make(map[string]cachedResponse),
	}
}

func (c *Client) SearchIssues(ctx context.Context, query, series, issue string) ([]Issue, error) {
	values := url.Values{}
	if series != "" {
		values.Set("series_name", series)
	} else if query != "" {
		values.Set("series_name", query)
	}
	if issue != "" {
		values.Set("number", issue)
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
	issue, _, err := c.GetIssueConditional(ctx, id, ConditionalRequest{})
	return issue, err
}

func (c *Client) GetIssueConditional(ctx context.Context, id int, conditional ConditionalRequest) (*Issue, FetchInfo, error) {
	var raw map[string]any
	info, err := c.getConditional(ctx, fmt.Sprintf("/issue/%d/", id), nil, conditional, &raw)
	if err != nil {
		return nil, info, err
	}
	if info.NotModified {
		return nil, info, nil
	}

	issue := issueFromMap(raw)
	return &issue, info, nil
}

func (c *Client) GetCharacter(ctx context.Context, id int) (*MetronCharacter, error) {
	character, _, err := c.GetCharacterConditional(ctx, id, ConditionalRequest{})
	return character, err
}

func (c *Client) GetCharacterConditional(ctx context.Context, id int, conditional ConditionalRequest) (*MetronCharacter, FetchInfo, error) {
	var raw map[string]any
	info, err := c.getConditional(ctx, fmt.Sprintf("/character/%d/", id), nil, conditional, &raw)
	if err != nil {
		return nil, info, err
	}
	if info.NotModified {
		return nil, info, nil
	}

	character := characterFromMap(raw)
	return &character, info, nil
}

func (c *Client) SearchCharacters(ctx context.Context, query string) ([]MetronCharacter, error) {
	values := url.Values{}
	if query != "" {
		values.Set("name", query)
	}

	results, err := c.getList(ctx, "/character/", values)
	if err != nil {
		return nil, err
	}
	characters := make([]MetronCharacter, 0, len(results))
	for _, raw := range results {
		characters = append(characters, characterFromMap(raw))
	}
	return characters, nil
}

func (c *Client) GetCharacterIssues(ctx context.Context, id int) ([]Issue, error) {
	results, err := c.getAllList(ctx, fmt.Sprintf("/character/%d/issue_list/", id), nil)
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, 0, len(results))
	for _, raw := range results {
		if issue := object(raw, "issue"); len(issue) > 0 {
			raw = issue
		}
		issues = append(issues, issueFromMap(raw))
	}
	return issues, nil
}

func (c *Client) EachCharacterIssuePage(ctx context.Context, id int, handle func([]Issue, int) error) error {
	next := fmt.Sprintf("/character/%d/issue_list/", id)
	var values url.Values
	for next != "" {
		page, err := c.getListPage(ctx, next, values)
		if err != nil {
			return err
		}

		issues := make([]Issue, 0, len(page.results))
		for _, raw := range page.results {
			if issue := object(raw, "issue"); len(issue) > 0 {
				raw = issue
			}
			issues = append(issues, issueFromMap(raw))
		}
		if err := handle(issues, page.count); err != nil {
			return err
		}

		next = page.next
		values = nil
	}
	return nil
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

func (c *Client) SearchArcs(ctx context.Context, query string) ([]MetronArc, error) {
	values := url.Values{}
	if query != "" {
		values.Set("name", query)
	}

	results, err := c.getList(ctx, "/arc/", values)
	if err != nil {
		return nil, err
	}
	arcs := make([]MetronArc, 0, len(results))
	for _, raw := range results {
		arcs = append(arcs, arcFromMap(raw))
	}
	return arcs, nil
}

type SeriesSearchOptions struct {
	Query     string
	YearBegan int
	Volume    int
}

func (c *Client) SearchSeries(ctx context.Context, options SeriesSearchOptions) ([]Series, error) {
	values := url.Values{}
	if options.Query != "" {
		values.Set("name", options.Query)
	}
	if options.YearBegan > 0 {
		values.Set("year_began", strconv.Itoa(options.YearBegan))
	}
	if options.Volume > 0 {
		values.Set("volume", strconv.Itoa(options.Volume))
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

func (c *Client) GetSeries(ctx context.Context, id int) (*Series, error) {
	series, _, err := c.GetSeriesConditional(ctx, id, ConditionalRequest{})
	return series, err
}

func (c *Client) GetSeriesConditional(ctx context.Context, id int, conditional ConditionalRequest) (*Series, FetchInfo, error) {
	var raw map[string]any
	info, err := c.getConditional(ctx, fmt.Sprintf("/series/%d/", id), nil, conditional, &raw)
	if err != nil {
		return nil, info, err
	}
	if info.NotModified {
		return nil, info, nil
	}
	series := seriesFromMap(raw)
	return &series, info, nil
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
	list, _, err := c.GetReadingListConditional(ctx, id, ConditionalRequest{})
	return list, err
}

func (c *Client) GetReadingListConditional(ctx context.Context, id int, conditional ConditionalRequest) (*ReadingList, FetchInfo, error) {
	var raw map[string]any
	info, err := c.getConditional(ctx, fmt.Sprintf("/reading_list/%d/", id), nil, conditional, &raw)
	if err != nil {
		return nil, info, err
	}
	if info.NotModified {
		return nil, info, nil
	}

	list := readingListFromMap(raw)
	issues, err := c.GetReadingListIssues(ctx, id)
	if err != nil {
		return nil, info, err
	}
	list.Issues = issues
	return &list, info, nil
}

func (c *Client) GetArc(ctx context.Context, id int) (*MetronArc, error) {
	arc, _, err := c.GetArcConditional(ctx, id, ConditionalRequest{})
	return arc, err
}

func (c *Client) GetArcConditional(ctx context.Context, id int, conditional ConditionalRequest) (*MetronArc, FetchInfo, error) {
	arc, info, err := c.GetArcMetadataConditional(ctx, id, conditional)
	if err != nil {
		return nil, info, err
	}
	if info.NotModified {
		return nil, info, nil
	}

	issues, err := c.GetArcIssues(ctx, id)
	if err != nil {
		return nil, info, err
	}
	arc.Issues = issues
	return arc, info, nil
}

func (c *Client) GetArcMetadata(ctx context.Context, id int) (*MetronArc, error) {
	arc, _, err := c.GetArcMetadataConditional(ctx, id, ConditionalRequest{})
	return arc, err
}

func (c *Client) GetArcMetadataConditional(ctx context.Context, id int, conditional ConditionalRequest) (*MetronArc, FetchInfo, error) {
	var raw map[string]any
	info, err := c.getConditional(ctx, fmt.Sprintf("/arc/%d/", id), nil, conditional, &raw)
	if err != nil {
		return nil, info, err
	}
	if info.NotModified {
		return nil, info, nil
	}

	arc := arcFromMap(raw)
	return &arc, info, nil
}

func (c *Client) GetReadingListIssues(ctx context.Context, id int) ([]Issue, error) {
	results, err := c.getAllList(ctx, fmt.Sprintf("/reading_list/%d/items/", id), nil)
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, 0, len(results))
	for _, raw := range results {
		if issue := object(raw, "issue"); len(issue) > 0 {
			issue = cloneMap(issue)
			if tags := stringList(raw, "issue_type", "type", "tag", "tags"); len(tags) > 0 {
				issue["tags"] = mergeStringLists(stringList(issue, "tags"), tags)
			}
			issues = append(issues, issueFromMap(issue))
		}
	}
	return issues, nil
}

func (c *Client) GetArcIssues(ctx context.Context, id int) ([]Issue, error) {
	results, err := c.getAllList(ctx, fmt.Sprintf("/arc/%d/issue_list/", id), nil)
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, 0, len(results))
	for _, raw := range results {
		if issue := object(raw, "issue"); len(issue) > 0 {
			raw = issue
		}
		issues = append(issues, issueFromMap(raw))
	}
	return issues, nil
}

func (c *Client) get(ctx context.Context, path string, values url.Values, target any) error {
	_, err := c.getConditional(ctx, path, values, ConditionalRequest{}, target)
	return err
}

func (c *Client) getConditional(ctx context.Context, path string, values url.Values, conditional ConditionalRequest, target any) (FetchInfo, error) {
	if err := c.waitForRateLimit(ctx); err != nil {
		return FetchInfo{}, err
	}
	if err := c.waitForMinInterval(ctx); err != nil {
		return FetchInfo{}, err
	}

	requestURL := c.requestURL(path)
	if len(values) > 0 {
		requestURL += "?" + values.Encode()
	}

	cacheKey := requestURL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return FetchInfo{}, err
	}
	c.authorize(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ComicHero/0.1")
	conditionalHeader := conditional.LastModified != ""
	if conditional.LastModified != "" && !conditional.Force {
		req.Header.Set("If-Modified-Since", conditional.LastModified)
	} else if !conditional.Force {
		if cached, ok := c.cached(cacheKey); ok && cached.lastModified != "" {
			req.Header.Set("If-Modified-Since", cached.lastModified)
			conditionalHeader = true
		}
	}

	started := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordRequest(req, 0, started, conditionalHeader, err.Error())
		return FetchInfo{}, err
	}
	defer resp.Body.Close()
	c.recordRequest(req, resp.StatusCode, started, conditionalHeader, "")

	if resp.StatusCode == http.StatusNotModified {
		c.updateRateLimit(resp.Header)
		info := FetchInfo{LastModified: req.Header.Get("If-Modified-Since"), NotModified: true}
		if conditional.LastModified == "" {
			if cached, ok := c.cached(cacheKey); ok {
				info.NotModified = false
				return info, json.Unmarshal(cached.body, target)
			}
		}
		return info, nil
	}

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
	lastModified := resp.Header.Get("Last-Modified")
	c.storeCache(cacheKey, lastModified, body)

	return FetchInfo{LastModified: lastModified}, json.Unmarshal(body, target)
}

func (c *Client) waitForMinInterval(ctx context.Context) error {
	c.intervalMu.Lock()
	defer c.intervalMu.Unlock()
	if c.minInterval <= 0 {
		return nil
	}
	if wait := time.Until(c.nextRequest); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	c.nextRequest = time.Now().Add(c.minInterval)
	return nil
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
	Count   int              `json:"count"`
}

type listPage struct {
	results []map[string]any
	next    string
	count   int
}

func issueFromMap(raw map[string]any) Issue {
	series := object(raw, "series")
	publisher := object(raw, "publisher")
	if len(publisher) == 0 {
		publisher = object(series, "publisher")
	}

	return Issue{
		ID:           intValue(raw, "id"),
		Title:        firstString(raw, "title", "name"),
		StoryNames:   stringList(raw, "name", "story_names", "storyNames"),
		SeriesID:     firstInt(intValue(raw, "series_id"), intValue(series, "id")),
		Series:       fallbackString(firstString(raw, "series_name", "series"), firstString(series, "name")),
		SeriesYear:   firstInt(intValue(raw, "series_year", "year_began", "year"), intValue(series, "year_began", "year")),
		SeriesVolume: firstInt(intValue(raw, "series_volume"), intValue(series, "volume")),
		Issue:        firstString(raw, "number", "issue"),
		Number:       firstString(raw, "number", "issue"),
		Publisher:    fallbackString(firstString(raw, "publisher_name", "publisher"), firstString(publisher, "name")),
		CoverDate:    firstString(raw, "cover_date", "store_date", "date"),
		StoreDate:    firstString(raw, "store_date"),
		CoverImage:   firstString(raw, "image", "cover", "cover_image", "image_url"),
		Description:  firstString(raw, "desc", "description", "synopsis"),
		Modified:     firstString(raw, "modified"),
		Tags:         stringList(raw, "tags", "tag", "issue_type"),
		Arcs:         arcsFromMap(raw),
		Characters:   charactersFromMap(raw),
	}
}

func arcsFromMap(raw map[string]any) []MetronArc {
	values, ok := raw["arcs"].([]any)
	if !ok {
		return nil
	}

	arcs := make([]MetronArc, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			arc := arcFromMap(item)
			if arc.ID > 0 || arc.Name != "" {
				arcs = append(arcs, arc)
			}
		}
	}
	return arcs
}

func charactersFromMap(raw map[string]any) []MetronCharacter {
	values, ok := raw["characters"].([]any)
	if !ok {
		return nil
	}

	characters := make([]MetronCharacter, 0, len(values))
	for _, value := range values {
		if item, ok := value.(map[string]any); ok {
			character := characterFromMap(item)
			if character.ID > 0 || character.Name != "" {
				characters = append(characters, character)
			}
		}
	}
	return characters
}

func characterFromMap(raw map[string]any) MetronCharacter {
	return MetronCharacter{
		ID:          intValue(raw, "id"),
		Name:        firstString(raw, "name"),
		Aliases:     stringList(raw, "alias", "aliases"),
		Description: firstString(raw, "desc", "description"),
		Image:       firstString(raw, "image"),
	}
}

func stringList(raw map[string]any, keys ...string) []string {
	seen := map[string]bool{}
	var values []string
	for _, key := range keys {
		switch rawValue := raw[key].(type) {
		case string:
			value := strings.TrimSpace(rawValue)
			if value != "" && !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		case []string:
			for _, item := range rawValue {
				value := strings.TrimSpace(item)
				if value == "" || seen[value] {
					continue
				}
				seen[value] = true
				values = append(values, value)
			}
		case []any:
			for _, item := range rawValue {
				value := stringValue(item)
				if value == "" || seen[value] {
					continue
				}
				seen[value] = true
				values = append(values, value)
			}
		case map[string]any:
			value := stringValue(rawValue)
			if value != "" && !seen[value] {
				seen[value] = true
				values = append(values, value)
			}
		}
	}
	return values
}

func mergeStringLists(lists ...[]string) []string {
	seen := map[string]bool{}
	values := []string{}
	for _, list := range lists {
		for _, item := range list {
			value := strings.TrimSpace(item)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
	}
	return values
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		return strings.TrimSpace(firstString(typed, "name", "title", "label", "value", "slug"))
	default:
		return ""
	}
}

func cloneMap(raw map[string]any) map[string]any {
	next := make(map[string]any, len(raw))
	for key, value := range raw {
		next[key] = value
	}
	return next
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
	user := object(raw, "user")
	list := ReadingList{
		ID:                intValue(raw, "id"),
		Name:              firstString(raw, "name", "title"),
		Slug:              firstString(raw, "slug"),
		User:              MetronUser{ID: intValue(user, "id"), Username: firstString(user, "username", "name")},
		ListType:          firstString(raw, "list_type", "listType", "type"),
		IsPrivate:         boolValue(raw, "is_private", "isPrivate"),
		AttributionSource: firstString(raw, "attribution_source", "attributionSource"),
		AttributionURL:    firstString(raw, "attribution_url", "attributionUrl"),
		AverageRating:     floatValue(raw, "average_rating", "averageRating"),
		RatingCount:       intValue(raw, "rating_count", "ratingCount"),
		Modified:          firstString(raw, "modified"),
		Image:             firstString(raw, "image"),
		ItemsURL:          firstString(raw, "items_url", "itemsUrl"),
		ResourceURL:       firstString(raw, "resource_url", "resourceUrl"),
		Description:       firstString(raw, "desc", "description", "summary"),
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

func floatValue(raw map[string]any, keys ...string) float64 {
	for _, key := range keys {
		switch value := raw[key].(type) {
		case float64:
			return value
		case float32:
			return float64(value)
		case int:
			return float64(value)
		case int64:
			return float64(value)
		case json.Number:
			parsed, _ := value.Float64()
			return parsed
		case string:
			parsed, _ := strconv.ParseFloat(value, 64)
			return parsed
		}
	}
	return 0
}

func boolValue(raw map[string]any, keys ...string) bool {
	for _, key := range keys {
		switch value := raw[key].(type) {
		case bool:
			return value
		case float64:
			return value != 0
		case string:
			parsed, _ := strconv.ParseBool(value)
			return parsed
		}
	}
	return false
}

func arcFromMap(raw map[string]any) MetronArc {
	arc := MetronArc{
		ID:          intValue(raw, "id"),
		Name:        firstString(raw, "name", "title"),
		Description: firstString(raw, "desc", "description", "summary"),
		Image:       firstString(raw, "image"),
		Modified:    firstString(raw, "modified"),
	}

	for _, key := range []string{"issues", "issue", "comics"} {
		values, ok := raw[key].([]any)
		if !ok {
			continue
		}
		for _, value := range values {
			if item, ok := value.(map[string]any); ok {
				arc.Issues = append(arc.Issues, issueFromMap(item))
			}
		}
	}
	return arc
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
