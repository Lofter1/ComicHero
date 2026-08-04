package metron

import (
	"context"
	"net/url"
	"strconv"
)

// PublisherList is the slim shape returned by the publisher list endpoint.
type PublisherList struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Modified string `json:"modified"`
}

// Publisher is the full publisher shape returned by the publisher detail
// endpoint.
type Publisher struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Founded     int    `json:"founded"`
	Country     string `json:"country"`
	Desc        string `json:"desc"`
	Image       string `json:"image"`
	CVID        int    `json:"cv_id"`
	GCDID       int    `json:"gcd_id"`
	ResourceURL string `json:"resource_url"`
	Modified    string `json:"modified"`
}

// PublisherListOptions filters GET /api/publisher/.
type PublisherListOptions struct {
	CVID       int
	GCDID      int
	ModifiedGT string
	Name       string
	Page       int
}

func (o PublisherListOptions) values() url.Values {
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

// PublisherService handles /api/publisher/ endpoints.
type PublisherService struct{ client *Client }

// List returns one page of publishers matching opts.
func (s *PublisherService) List(ctx context.Context, opts PublisherListOptions) (PagedResponse[PublisherList], error) {
	return listPage[PublisherList](ctx, s.client, "/publisher/", opts.values())
}

// All walks every page of publishers matching opts.
func (s *PublisherService) All(ctx context.Context, opts PublisherListOptions) ([]PublisherList, error) {
	return listAll[PublisherList](ctx, s.client, "/publisher/", opts.values())
}

// Get returns a single publisher by its Metron ID.
func (s *PublisherService) Get(ctx context.Context, id int) (*Publisher, error) {
	var publisher Publisher
	if err := s.client.get(ctx, "/publisher/"+strconv.Itoa(id)+"/", nil, &publisher); err != nil {
		return nil, err
	}
	return &publisher, nil
}

// SeriesList returns one page of the series published by publisher id.
func (s *PublisherService) SeriesList(ctx context.Context, id int, page int) (PagedResponse[SeriesList], error) {
	return listPage[SeriesList](ctx, s.client, "/publisher/"+strconv.Itoa(id)+"/series_list/", setPage(nil, page))
}

// AllSeries walks every page of the series published by publisher id.
func (s *PublisherService) AllSeries(ctx context.Context, id int) ([]SeriesList, error) {
	return listAll[SeriesList](ctx, s.client, "/publisher/"+strconv.Itoa(id)+"/series_list/", nil)
}
