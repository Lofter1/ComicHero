package metron

import (
	"context"
	"net/url"
	"strconv"
)

// CreatorList is the slim shape returned by the creator list endpoint, and
// the shape embedded in other resources (e.g. Character.Creators).
type CreatorList struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Modified string `json:"modified"`
}

// Creator is the full creator shape returned by the creator detail
// endpoint.
type Creator struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Birth       string   `json:"birth"`
	Death       string   `json:"death"`
	Desc        string   `json:"desc"`
	Image       string   `json:"image"`
	Alias       []string `json:"alias"`
	CVID        int      `json:"cv_id"`
	GCDID       int      `json:"gcd_id"`
	ResourceURL string   `json:"resource_url"`
	Modified    string   `json:"modified"`
}

// CreatorListOptions filters GET /api/creator/.
type CreatorListOptions struct {
	CVID       int
	GCDID      int
	ModifiedGT string
	Name       string
	Page       int
}

func (o CreatorListOptions) values() url.Values {
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

// CreatorService handles /api/creator/ endpoints.
type CreatorService struct{ client *Client }

// List returns one page of creators matching opts.
func (s *CreatorService) List(ctx context.Context, opts CreatorListOptions) (PagedResponse[CreatorList], error) {
	return listPage[CreatorList](ctx, s.client, "/creator/", opts.values())
}

// All walks every page of creators matching opts.
func (s *CreatorService) All(ctx context.Context, opts CreatorListOptions) ([]CreatorList, error) {
	return listAll[CreatorList](ctx, s.client, "/creator/", opts.values())
}

// Get returns a single creator by its Metron ID.
func (s *CreatorService) Get(ctx context.Context, id int) (*Creator, error) {
	var creator Creator
	if err := s.client.get(ctx, "/creator/"+strconv.Itoa(id)+"/", nil, &creator); err != nil {
		return nil, err
	}
	return &creator, nil
}
