package metron

import (
	"context"
	"net/url"
	"strconv"
)

// IssueList is the slim shape returned by issue list endpoints (issue
// search, and every resource's issue_list sub-endpoint).
type IssueList struct {
	ID        int             `json:"id"`
	Series    IssueListSeries `json:"series"`
	Number    string          `json:"number"`
	Issue     string          `json:"issue"`
	CoverDate string          `json:"cover_date"`
	StoreDate string          `json:"store_date"`
	Image     string          `json:"image"`
	CoverHash string          `json:"cover_hash"`
	Modified  string          `json:"modified"`
}

// Issue is the full issue shape returned by the issue detail endpoint.
//
// Note: CoverHash is a perceptual hash created with ImageHash
// (https://github.com/JohannesBuchner/imagehash).
type Issue struct {
	ID            int             `json:"id"`
	Publisher     BasicPublisher  `json:"publisher"`
	Imprint       BasicImprint    `json:"imprint"`
	Series        IssueSeries     `json:"series"`
	Number        string          `json:"number"`
	AltNumber     string          `json:"alt_number"`
	Title         string          `json:"title"`
	Name          []string        `json:"name"`
	CoverDate     string          `json:"cover_date"`
	StoreDate     string          `json:"store_date"`
	FOCDate       string          `json:"foc_date"`
	Price         Price           `json:"price"`
	PriceCurrency string          `json:"price_currency"`
	Rating        Rating          `json:"rating"`
	SKU           string          `json:"sku"`
	ISBN          string          `json:"isbn"`
	UPC           string          `json:"upc"`
	Page          int             `json:"page"`
	Desc          string          `json:"desc"`
	Image         string          `json:"image"`
	CoverHash     string          `json:"cover_hash"`
	AverageRating float64         `json:"average_rating"`
	RatingCount   int             `json:"rating_count"`
	Arcs          []ArcList       `json:"arcs"`
	Credits       []Credit        `json:"credits"`
	Characters    []CharacterList `json:"characters"`
	Teams         []TeamList      `json:"teams"`
	Universes     []UniverseList  `json:"universes"`
	Reprints      []Reprint       `json:"reprints"`
	Variants      []VariantIssue  `json:"variants"`
	CVID          int             `json:"cv_id"`
	GCDID         int             `json:"gcd_id"`
	ResourceURL   string          `json:"resource_url"`
	Modified      string          `json:"modified"`
}

// IssueListOptions filters GET /api/issue/.
type IssueListOptions struct {
	AltNumber            string
	CharacterID          int
	CoverHash            string
	CoverMonth           int
	CoverYear            int
	CreatorID            int
	CVID                 int
	FOCDate              string
	FOCDateRangeAfter    string
	FOCDateRangeBefore   string
	GCDID                int
	ImprintID            int
	ImprintName          string
	MissingCVID          *bool
	MissingGCDID         *bool
	ModifiedGT           string
	Number               string
	Page                 int
	PublisherID          int
	PublisherName        string
	Rating               string
	RoleIDs              []int
	SeriesAltNames       string
	SeriesID             int
	SeriesName           string
	SeriesQuery          string
	SeriesVolume         int
	SeriesYearBegan      int
	SKU                  string
	StoreDate            string
	StoreDateRangeAfter  string
	StoreDateRangeBefore string
	TeamID               int
	UniverseID           int
	UPC                  string
	UPCStartsWith        string
}

func (o IssueListOptions) values() url.Values {
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
	setStr("alt_number", o.AltNumber)
	setInt("character_id", o.CharacterID)
	setStr("cover_hash", o.CoverHash)
	setInt("cover_month", o.CoverMonth)
	setInt("cover_year", o.CoverYear)
	setInt("creator_id", o.CreatorID)
	setInt("cv_id", o.CVID)
	setStr("foc_date", o.FOCDate)
	setStr("foc_date_range_after", o.FOCDateRangeAfter)
	setStr("foc_date_range_before", o.FOCDateRangeBefore)
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
	setStr("number", o.Number)
	setInt("publisher_id", o.PublisherID)
	setStr("publisher_name", o.PublisherName)
	setStr("rating", o.Rating)
	if len(o.RoleIDs) > 0 {
		values.Set("role_id", joinInts(o.RoleIDs))
	}
	setStr("series_alt_names", o.SeriesAltNames)
	setInt("series_id", o.SeriesID)
	setStr("series_name", o.SeriesName)
	setStr("series_q", o.SeriesQuery)
	setInt("series_volume", o.SeriesVolume)
	setInt("series_year_began", o.SeriesYearBegan)
	setStr("sku", o.SKU)
	setStr("store_date", o.StoreDate)
	setStr("store_date_range_after", o.StoreDateRangeAfter)
	setStr("store_date_range_before", o.StoreDateRangeBefore)
	setInt("team_id", o.TeamID)
	setInt("universe_id", o.UniverseID)
	setStr("upc", o.UPC)
	setStr("upc_starts_with", o.UPCStartsWith)
	return setPage(values, o.Page)
}

// IssueService handles /api/issue/ endpoints.
type IssueService struct{ client *Client }

// List returns one page of issues matching opts.
func (s *IssueService) List(ctx context.Context, opts IssueListOptions) (PagedResponse[IssueList], error) {
	return listPage[IssueList](ctx, s.client, "/issue/", opts.values())
}

// All walks every page of issues matching opts.
func (s *IssueService) All(ctx context.Context, opts IssueListOptions) ([]IssueList, error) {
	return listAll[IssueList](ctx, s.client, "/issue/", opts.values())
}

// Get returns a single issue by its Metron ID.
func (s *IssueService) Get(ctx context.Context, id int) (*Issue, error) {
	var issue Issue
	if err := s.client.get(ctx, "/issue/"+strconv.Itoa(id)+"/", nil, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}
