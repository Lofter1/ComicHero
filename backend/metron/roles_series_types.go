package metron

import (
	"context"
	"net/url"
)

// RoleListOptions filters GET /api/role/.
type RoleListOptions struct {
	ModifiedGT string
	Name       string
	Page       int
}

func (o RoleListOptions) values() url.Values {
	values := url.Values{}
	if o.ModifiedGT != "" {
		values.Set("modified_gt", o.ModifiedGT)
	}
	if o.Name != "" {
		values.Set("name", o.Name)
	}
	return setPage(values, o.Page)
}

// RoleService handles /api/role/ endpoints.
type RoleService struct{ client *Client }

// List returns one page of creator roles matching opts.
func (s *RoleService) List(ctx context.Context, opts RoleListOptions) (PagedResponse[Role], error) {
	return listPage[Role](ctx, s.client, "/role/", opts.values())
}

// All walks every page of creator roles matching opts.
func (s *RoleService) All(ctx context.Context, opts RoleListOptions) ([]Role, error) {
	return listAll[Role](ctx, s.client, "/role/", opts.values())
}

// SeriesTypeListOptions filters GET /api/series_type/.
type SeriesTypeListOptions struct {
	ModifiedGT string
	Name       string
	Page       int
}

func (o SeriesTypeListOptions) values() url.Values {
	values := url.Values{}
	if o.ModifiedGT != "" {
		values.Set("modified_gt", o.ModifiedGT)
	}
	if o.Name != "" {
		values.Set("name", o.Name)
	}
	return setPage(values, o.Page)
}

// SeriesTypeService handles /api/series_type/ endpoints.
type SeriesTypeService struct{ client *Client }

// List returns one page of series types matching opts.
func (s *SeriesTypeService) List(ctx context.Context, opts SeriesTypeListOptions) (PagedResponse[SeriesTypeRef], error) {
	return listPage[SeriesTypeRef](ctx, s.client, "/series_type/", opts.values())
}

// All walks every page of series types matching opts.
func (s *SeriesTypeService) All(ctx context.Context, opts SeriesTypeListOptions) ([]SeriesTypeRef, error) {
	return listAll[SeriesTypeRef](ctx, s.client, "/series_type/", opts.values())
}
