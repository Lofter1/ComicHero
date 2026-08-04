package metron

import (
	"context"
	"net/url"
	"strconv"
)

// SeriesList is the slim shape returned by the series list endpoint.
type SeriesList struct {
	ID         int    `json:"id"`
	Series     string `json:"series"`
	YearBegan  int    `json:"year_began"`
	YearEnd    int    `json:"year_end"`
	Volume     int    `json:"volume"`
	IssueCount int    `json:"issue_count"`
	Modified   string `json:"modified"`
}

// IssueSeries is the shape of a series as embedded in a full Issue.
type IssueSeries struct {
	ID         int           `json:"id"`
	Name       string        `json:"name"`
	AltNames   []string      `json:"alt_names"`
	SortName   string        `json:"sort_name"`
	Volume     int           `json:"volume"`
	YearBegan  int           `json:"year_began"`
	SeriesType SeriesTypeRef `json:"series_type"`
	Genres     []Genre       `json:"genres"`
}

// IssueListSeries is the shape of a series as embedded in an IssueList.
type IssueListSeries struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Volume    int    `json:"volume"`
	YearBegan int    `json:"year_began"`
}

// Series is the full series shape returned by the series detail endpoint.
type Series struct {
	ID          int                `json:"id"`
	Name        string             `json:"name"`
	SortName    string             `json:"sort_name"`
	AltNames    []string           `json:"alt_names"`
	Volume      int                `json:"volume"`
	SeriesType  SeriesTypeRef      `json:"series_type"`
	Status      string             `json:"status"`
	Publisher   BasicPublisher     `json:"publisher"`
	Imprint     BasicImprint       `json:"imprint"`
	YearBegan   int                `json:"year_began"`
	YearEnd     int                `json:"year_end"`
	Desc        string             `json:"desc"`
	IssueCount  int                `json:"issue_count"`
	Genres      []Genre            `json:"genres"`
	Associated  []AssociatedSeries `json:"associated"`
	CVID        int                `json:"cv_id"`
	GCDID       int                `json:"gcd_id"`
	ResourceURL string             `json:"resource_url"`
	Modified    string             `json:"modified"`
}

// SeriesListOptions filters GET /api/series/.
type SeriesListOptions struct {
	AltNames      string
	CharacterID   int
	CreatorID     int
	CVID          int
	GCDID         int
	ImprintID     int
	ImprintName   string
	MissingCVID   *bool
	MissingGCDID  *bool
	ModifiedGT    string
	Name          string
	Page          int
	PublisherID   int
	PublisherName string
	Query         string
	RoleIDs       []int
	SeriesType    string
	SeriesTypeID  int
	Status        int
	TeamID        int
	UniverseID    int
	Volume        int
	YearBegan     int
	YearEnd       int
}

func (o SeriesListOptions) values() url.Values {
	values := url.Values{}
	setStr := func(key, value string) {
		if value != "" {
			values.Set(key, value)
		}
	}
	setInt := func(key string, value int) {
		if value > 0 {
			values.Set(key, strconv.Itoa(value))
		}
	}
	setStr("alt_names", o.AltNames)
	setInt("character_id", o.CharacterID)
	setInt("creator_id", o.CreatorID)
	setInt("cv_id", o.CVID)
	setInt("gcd_id", o.GCDID)
	setInt("imprint_id", o.ImprintID)
	setStr("imprint_name", o.ImprintName)
	if o.MissingCVID != nil {
		values.Set("missing_cv_id", strconv.FormatBool(*o.MissingCVID))
	}
	if o.MissingGCDID != nil {
		values.Set("missing_gcd_id", strconv.FormatBool(*o.MissingGCDID))
	}
	setStr("modified_gt", o.ModifiedGT)
	setStr("name", o.Name)
	setInt("publisher_id", o.PublisherID)
	setStr("publisher_name", o.PublisherName)
	setStr("q", o.Query)
	if len(o.RoleIDs) > 0 {
		values.Set("role_id", joinInts(o.RoleIDs))
	}
	setStr("series_type", o.SeriesType)
	setInt("series_type_id", o.SeriesTypeID)
	if o.Status > 0 {
		values.Set("status", strconv.Itoa(o.Status))
	}
	setInt("team_id", o.TeamID)
	setInt("universe_id", o.UniverseID)
	setInt("volume", o.Volume)
	setInt("year_began", o.YearBegan)
	setInt("year_end", o.YearEnd)
	return setPage(values, o.Page)
}

func joinInts(values []int) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += ","
		}
		out += strconv.Itoa(v)
	}
	return out
}

// SeriesService handles /api/series/ endpoints.
type SeriesService struct{ client *Client }

// List returns one page of series matching opts.
func (s *SeriesService) List(ctx context.Context, opts SeriesListOptions) (PagedResponse[SeriesList], error) {
	return listPage[SeriesList](ctx, s.client, "/series/", opts.values())
}

// All walks every page of series matching opts.
func (s *SeriesService) All(ctx context.Context, opts SeriesListOptions) ([]SeriesList, error) {
	return listAll[SeriesList](ctx, s.client, "/series/", opts.values())
}

// Get returns a single series by its Metron ID.
func (s *SeriesService) Get(ctx context.Context, id int) (*Series, error) {
	var series Series
	if err := s.client.get(ctx, "/series/"+strconv.Itoa(id)+"/", nil, &series); err != nil {
		return nil, err
	}
	return &series, nil
}

// IssueList returns one page of the issues in series id.
func (s *SeriesService) IssueList(ctx context.Context, id int, page int) (PagedResponse[IssueList], error) {
	return listPage[IssueList](ctx, s.client, "/series/"+strconv.Itoa(id)+"/issue_list/", setPage(nil, page))
}

// AllIssues walks every page of the issues in series id.
func (s *SeriesService) AllIssues(ctx context.Context, id int) ([]IssueList, error) {
	return listAll[IssueList](ctx, s.client, "/series/"+strconv.Itoa(id)+"/issue_list/", nil)
}
