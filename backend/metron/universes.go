package metron

import (
	"context"
	"net/url"
	"strconv"
)

// UniverseList is the slim shape returned by the universe list endpoint,
// and the shape embedded in other resources (e.g. Character.Universes).
type UniverseList struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Modified string `json:"modified"`
}

// Universe is the full universe shape returned by the universe detail
// endpoint.
type Universe struct {
	ID          int            `json:"id"`
	Publisher   BasicPublisher `json:"publisher"`
	Name        string         `json:"name"`
	Designation string         `json:"designation"`
	Desc        string         `json:"desc"`
	GCDID       int            `json:"gcd_id"`
	Image       string         `json:"image"`
	ResourceURL string         `json:"resource_url"`
	Modified    string         `json:"modified"`
}

// UniverseListOptions filters GET /api/universe/.
type UniverseListOptions struct {
	Designation string
	ModifiedGT  string
	Name        string
	Page        int
}

func (o UniverseListOptions) values() url.Values {
	values := url.Values{}
	if o.Designation != "" {
		values.Set("designation", o.Designation)
	}
	if o.ModifiedGT != "" {
		values.Set("modified_gt", o.ModifiedGT)
	}
	if o.Name != "" {
		values.Set("name", o.Name)
	}
	return setPage(values, o.Page)
}

// UniverseService handles /api/universe/ endpoints.
type UniverseService struct{ client *Client }

// List returns one page of universes matching opts.
func (s *UniverseService) List(ctx context.Context, opts UniverseListOptions) (PagedResponse[UniverseList], error) {
	return listPage[UniverseList](ctx, s.client, "/universe/", opts.values())
}

// All walks every page of universes matching opts.
func (s *UniverseService) All(ctx context.Context, opts UniverseListOptions) ([]UniverseList, error) {
	return listAll[UniverseList](ctx, s.client, "/universe/", opts.values())
}

// Get returns a single universe by its Metron ID.
func (s *UniverseService) Get(ctx context.Context, id int) (*Universe, error) {
	var universe Universe
	if err := s.client.get(ctx, "/universe/"+strconv.Itoa(id)+"/", nil, &universe); err != nil {
		return nil, err
	}
	return &universe, nil
}
