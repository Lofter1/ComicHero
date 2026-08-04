package metron

import (
	"context"
	"net/url"
	"strconv"
)

// PullListIssue is an issue on a pull-list series, i.e. an upcoming or
// recent issue of a series the user follows.
type PullListIssue struct {
	ID        int             `json:"id"`
	Series    IssueListSeries `json:"series"`
	Number    string          `json:"number"`
	Issue     string          `json:"issue"`
	CoverDate string          `json:"cover_date"`
	StoreDate string          `json:"store_date"`
	Image     string          `json:"image"`
	Modified  string          `json:"modified"`
}

// PullList is the authenticated user's pull list summary.
type PullList struct {
	ID          int    `json:"id"`
	SeriesCount int    `json:"series_count"`
	SeriesURL   string `json:"series_url"`
	Modified    string `json:"modified"`
}

// PullListSeries is a single series on the authenticated user's pull list.
type PullListSeries struct {
	ID      int        `json:"id"`
	Series  SeriesList `json:"series"`
	AddedOn string     `json:"added_on"`
}

// PullListIssuesOptions filters GET /api/pull_list/issues/.
type PullListIssuesOptions struct {
	Page            int
	StoreDateAfter  string
	StoreDateBefore string
}

func (o PullListIssuesOptions) values() url.Values {
	values := url.Values{}
	if o.StoreDateAfter != "" {
		values.Set("store_date_after", o.StoreDateAfter)
	}
	if o.StoreDateBefore != "" {
		values.Set("store_date_before", o.StoreDateBefore)
	}
	return setPage(values, o.Page)
}

// PullListService handles /api/pull_list/ endpoints.
type PullListService struct{ client *Client }

// List returns one page of the authenticated user's pull lists.
func (s *PullListService) List(ctx context.Context, page int) (PagedResponse[PullList], error) {
	return listPage[PullList](ctx, s.client, "/pull_list/", setPage(nil, page))
}

// Get returns a single pull list by its Metron ID.
func (s *PullListService) Get(ctx context.Context, id int) (*PullList, error) {
	var list PullList
	if err := s.client.get(ctx, "/pull_list/"+strconv.Itoa(id)+"/", nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// Series returns one page of the authenticated user's pull-list series.
func (s *PullListService) Series(ctx context.Context, page int) (PagedResponse[PullListSeries], error) {
	return listPage[PullListSeries](ctx, s.client, "/pull_list/series/", setPage(nil, page))
}

// AllSeries walks every page of the authenticated user's pull-list series.
func (s *PullListService) AllSeries(ctx context.Context) ([]PullListSeries, error) {
	return listAll[PullListSeries](ctx, s.client, "/pull_list/series/", nil)
}

// AddSeries adds seriesID to the authenticated user's pull list. Metron
// takes the target series as a query parameter rather than a JSON body.
func (s *PullListService) AddSeries(ctx context.Context, seriesID int) (*PullListSeries, error) {
	var added PullListSeries
	path := "/pull_list/series/add/?series_id=" + strconv.Itoa(seriesID)
	if err := s.client.post(ctx, path, nil, &added); err != nil {
		return nil, err
	}
	return &added, nil
}

// RemoveSeries removes seriesID from the authenticated user's pull list.
func (s *PullListService) RemoveSeries(ctx context.Context, seriesID int) error {
	return s.client.delete(ctx, "/pull_list/series/"+strconv.Itoa(seriesID)+"/remove/")
}

// Issues returns one page of issues for series on the authenticated
// user's pull list, optionally filtered by store date.
func (s *PullListService) Issues(ctx context.Context, opts PullListIssuesOptions) (PagedResponse[PullListIssue], error) {
	return listPage[PullListIssue](ctx, s.client, "/pull_list/issues/", opts.values())
}

// AllIssues walks every page of issues for series on the authenticated
// user's pull list, optionally filtered by store date.
func (s *PullListService) AllIssues(ctx context.Context, opts PullListIssuesOptions) ([]PullListIssue, error) {
	return listAll[PullListIssue](ctx, s.client, "/pull_list/issues/", opts.values())
}
