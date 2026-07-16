package metron

import (
	"context"
	"fmt"
	"net/url"
)

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
