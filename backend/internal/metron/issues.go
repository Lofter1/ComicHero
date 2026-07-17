package metron

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

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

func (c *Client) SearchIssuesByComicVineID(ctx context.Context, comicVineID int) ([]Issue, error) {
	values := url.Values{}
	values.Set("cv_id", strconv.Itoa(comicVineID))

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

type IssueModifiedSearchOptions struct {
	ModifiedAfter string
	PublisherName string
	SeriesName    string
}

// SearchModifiedIssues returns every page from the issue list endpoint. It
// deliberately uses list payloads only and never expands issue details.
func (c *Client) SearchModifiedIssues(ctx context.Context, options IssueModifiedSearchOptions) ([]Issue, error) {
	values := url.Values{}
	values.Set("modified_gt", options.ModifiedAfter)
	if options.PublisherName != "" {
		values.Set("publisher_name", options.PublisherName)
	}
	if options.SeriesName != "" {
		values.Set("series_name", options.SeriesName)
	}
	results, err := c.getAllList(ctx, "/issue/", values)
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(results))
	for _, raw := range results {
		issues = append(issues, issueFromMap(raw))
	}
	return issues, nil
}
