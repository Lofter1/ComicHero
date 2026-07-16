package metron

import (
	"context"
	"fmt"
	"net/url"
)

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

// ListReadingLists returns every page from the unfiltered reading-list endpoint.
func (c *Client) ListReadingLists(ctx context.Context) ([]ReadingList, error) {
	results, err := c.getAllList(ctx, "/reading_list/", nil)
	if err != nil {
		return nil, err
	}
	lists := make([]ReadingList, 0, len(results))
	for _, raw := range results {
		lists = append(lists, readingListFromMap(raw))
	}
	return lists, nil
}

// SearchModifiedReadingLists returns every reading-list page modified after
// the supplied timestamp without expanding list items.
func (c *Client) SearchModifiedReadingLists(ctx context.Context, modifiedAfter string) ([]ReadingList, error) {
	values := url.Values{}
	values.Set("modified_gt", modifiedAfter)
	results, err := c.getAllList(ctx, "/reading_list/", values)
	if err != nil {
		return nil, err
	}
	lists := make([]ReadingList, 0, len(results))
	for _, raw := range results {
		lists = append(lists, readingListFromMap(raw))
	}
	return lists, nil
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
