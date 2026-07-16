package metron

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

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
