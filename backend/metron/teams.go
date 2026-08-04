package metron

import (
	"context"
	"net/url"
	"strconv"
)

// TeamList is the slim shape returned by the team list endpoint, and the
// shape embedded in other resources (e.g. Character.Teams).
type TeamList struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Modified string `json:"modified"`
}

// Team is the full team shape returned by the team detail endpoint.
type Team struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Desc        string         `json:"desc"`
	Image       string         `json:"image"`
	Creators    []CreatorList  `json:"creators"`
	Universes   []UniverseList `json:"universes"`
	CVID        int            `json:"cv_id"`
	GCDID       int            `json:"gcd_id"`
	ResourceURL string         `json:"resource_url"`
	Modified    string         `json:"modified"`
}

// TeamListOptions filters GET /api/team/.
type TeamListOptions struct {
	CVID       int
	GCDID      int
	ModifiedGT string
	Name       string
	Page       int
}

func (o TeamListOptions) values() url.Values {
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

// TeamService handles /api/team/ endpoints.
type TeamService struct{ client *Client }

// List returns one page of teams matching opts.
func (s *TeamService) List(ctx context.Context, opts TeamListOptions) (PagedResponse[TeamList], error) {
	return listPage[TeamList](ctx, s.client, "/team/", opts.values())
}

// All walks every page of teams matching opts.
func (s *TeamService) All(ctx context.Context, opts TeamListOptions) ([]TeamList, error) {
	return listAll[TeamList](ctx, s.client, "/team/", opts.values())
}

// Get returns a single team by its Metron ID.
func (s *TeamService) Get(ctx context.Context, id int) (*Team, error) {
	var team Team
	if err := s.client.get(ctx, "/team/"+strconv.Itoa(id)+"/", nil, &team); err != nil {
		return nil, err
	}
	return &team, nil
}

// IssueList returns one page of the issues team id appears in.
func (s *TeamService) IssueList(ctx context.Context, id int, page int) (PagedResponse[IssueList], error) {
	return listPage[IssueList](ctx, s.client, "/team/"+strconv.Itoa(id)+"/issue_list/", setPage(nil, page))
}

// AllIssues walks every page of the issues team id appears in.
func (s *TeamService) AllIssues(ctx context.Context, id int) ([]IssueList, error) {
	return listAll[IssueList](ctx, s.client, "/team/"+strconv.Itoa(id)+"/issue_list/", nil)
}
