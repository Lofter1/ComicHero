package metron

import (
	"context"
	"net/url"
	"strconv"
)

// ImprintList is the slim shape returned by the imprint list endpoint.
type ImprintList struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Modified string `json:"modified"`
}

// Imprint is the full imprint shape returned by the imprint detail
// endpoint.
type Imprint struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Founded     int            `json:"founded"`
	Desc        string         `json:"desc"`
	Image       string         `json:"image"`
	CVID        int            `json:"cv_id"`
	GCDID       int            `json:"gcd_id"`
	Publisher   BasicPublisher `json:"publisher"`
	ResourceURL string         `json:"resource_url"`
	Modified    string         `json:"modified"`
}

// ImprintListOptions filters GET /api/imprint/.
type ImprintListOptions struct {
	CVID       int
	GCDID      int
	ModifiedGT string
	Name       string
	Page       int
}

func (o ImprintListOptions) values() url.Values {
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

// ImprintService handles /api/imprint/ endpoints.
type ImprintService struct{ client *Client }

// List returns one page of imprints matching opts.
func (s *ImprintService) List(ctx context.Context, opts ImprintListOptions) (PagedResponse[ImprintList], error) {
	return listPage[ImprintList](ctx, s.client, "/imprint/", opts.values())
}

// All walks every page of imprints matching opts.
func (s *ImprintService) All(ctx context.Context, opts ImprintListOptions) ([]ImprintList, error) {
	return listAll[ImprintList](ctx, s.client, "/imprint/", opts.values())
}

// Get returns a single imprint by its Metron ID.
func (s *ImprintService) Get(ctx context.Context, id int) (*Imprint, error) {
	var imprint Imprint
	if err := s.client.get(ctx, "/imprint/"+strconv.Itoa(id)+"/", nil, &imprint); err != nil {
		return nil, err
	}
	return &imprint, nil
}
