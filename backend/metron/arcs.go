package metron

import (
	"context"
	"net/url"
	"strconv"
)

// Arc is a Metron story arc.
type Arc struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	Image       string `json:"image"`
	CVID        int    `json:"cv_id"`
	GCDID       int    `json:"gcd_id"`
	ResourceURL string `json:"resource_url"`
	Modified    string `json:"modified"`
}

// ArcList is the slim shape returned by the arc list endpoint.
type ArcList struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Modified string `json:"modified"`
}

// ArcListOptions filters GET /api/arc/.
type ArcListOptions struct {
	CVID       int
	GCDID      int
	ModifiedGT string
	Name       string
	Page       int
}

func (o ArcListOptions) values() url.Values {
	values := url.Values{}
	if o.CVID > 0 {
		values.Set("cv_id", strconv.Itoa(o.CVID))
	}
	if o.GCDID > 0 {
		values.Set("gcd_id", strconv.Itoa(o.GCDID))
	}
	if o.ModifiedGT != "" {
		values.Set("modified_gt", o.ModifiedGT)
	}
	if o.Name != "" {
		values.Set("name", o.Name)
	}
	return setPage(values, o.Page)
}

// ArcService handles /api/arc/ endpoints.
type ArcService struct{ client *Client }

// List returns one page of arcs matching opts.
func (s *ArcService) List(ctx context.Context, opts ArcListOptions) (PagedResponse[ArcList], error) {
	return listPage[ArcList](ctx, s.client, "/arc/", opts.values())
}

// All walks every page of arcs matching opts and returns them concatenated.
func (s *ArcService) All(ctx context.Context, opts ArcListOptions) ([]ArcList, error) {
	return listAll[ArcList](ctx, s.client, "/arc/", opts.values())
}

// Get returns a single arc by its Metron ID.
func (s *ArcService) Get(ctx context.Context, id int) (*Arc, error) {
	var arc Arc
	if err := s.client.get(ctx, "/arc/"+strconv.Itoa(id)+"/", nil, &arc); err != nil {
		return nil, err
	}
	return &arc, nil
}

// IssueList returns one page of the issues belonging to arc id, in the
// order Metron itself considers correct for reading the arc.
func (s *ArcService) IssueList(ctx context.Context, id int, page int) (PagedResponse[IssueList], error) {
	return listPage[IssueList](ctx, s.client, "/arc/"+strconv.Itoa(id)+"/issue_list/", setPage(nil, page))
}

// AllIssues walks every page of arc id's issue list and returns every
// issue concatenated, preserving Metron's reading order.
func (s *ArcService) AllIssues(ctx context.Context, id int) ([]IssueList, error) {
	return listAll[IssueList](ctx, s.client, "/arc/"+strconv.Itoa(id)+"/issue_list/", nil)
}
