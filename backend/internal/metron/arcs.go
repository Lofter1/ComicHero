package metron

import (
	"context"
	"fmt"
	"net/url"
)

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
