package metron

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// BookFormat is the physical/digital format of a collected issue.
type BookFormat string

const (
	BookFormatBoth    BookFormat = "BOTH"
	BookFormatDigital BookFormat = "DIGITAL"
	BookFormatPrint   BookFormat = "PRINT"
)

// GradingCompany is a professional comic-grading company.
type GradingCompany string

const (
	GradingCompanyCBCS GradingCompany = "CBCS"
	GradingCompanyCGC  GradingCompany = "CGC"
	GradingCompanyPGX  GradingCompany = "PGX"
)

// ReadDate is a single recorded read of a collected issue.
type ReadDate struct {
	ID        int    `json:"id"`
	ReadDate  string `json:"read_date"`
	CreatedOn string `json:"created_on"`
}

// CollectionIssue is the slim issue shape embedded in collection items.
type CollectionIssue struct {
	ID        int             `json:"id"`
	Series    IssueListSeries `json:"series"`
	Number    string          `json:"number"`
	CoverDate string          `json:"cover_date"`
	StoreDate string          `json:"store_date"`
	Modified  string          `json:"modified"`
}

// CollectionList is the slim shape returned by the collection list
// endpoint.
type CollectionList struct {
	ID             int             `json:"id"`
	User           User            `json:"user"`
	Issue          CollectionIssue `json:"issue"`
	Quantity       int             `json:"quantity"`
	BookFormat     BookFormat      `json:"book_format"`
	Grade          *float64        `json:"grade"`
	GradingCompany string          `json:"grading_company"`
	PurchaseDate   string          `json:"purchase_date"`
	IsRead         bool            `json:"is_read"`
	ReadDates      []ReadDate      `json:"read_dates"`
	ReadCount      int             `json:"read_count"`
	Rating         *int            `json:"rating"`
	Modified       string          `json:"modified"`
}

// Collection is the full collection-item shape returned by the collection
// detail endpoint.
type Collection struct {
	ID              int             `json:"id"`
	User            User            `json:"user"`
	Issue           CollectionIssue `json:"issue"`
	Quantity        int             `json:"quantity"`
	BookFormat      BookFormat      `json:"book_format"`
	Grade           *float64        `json:"grade"`
	GradingCompany  string          `json:"grading_company"`
	PurchaseDate    string          `json:"purchase_date"`
	PurchasePrice   string          `json:"purchase_price"`
	PurchaseStore   string          `json:"purchase_store"`
	StorageLocation string          `json:"storage_location"`
	Notes           string          `json:"notes"`
	IsRead          bool            `json:"is_read"`
	DateRead        string          `json:"date_read"`
	ReadDates       []ReadDate      `json:"read_dates"`
	ReadCount       int             `json:"read_count"`
	Rating          *int            `json:"rating"`
	ResourceURL     string          `json:"resource_url"`
	CreatedOn       string          `json:"created_on"`
	Modified        string          `json:"modified"`
}

// CollectionRatingUpdate is the request/response body for updating the
// rating on a collection item. Read-tracking (is_read/date_read) is
// intentionally not editable here; use Scrobble instead.
type CollectionRatingUpdate struct {
	Rating *int `json:"rating"`
}

// CollectionListOptions filters GET /api/collection/.
type CollectionListOptions struct {
	BookFormat      BookFormat
	DateRead        string
	DateReadGT      string
	DateReadGTE     string
	DateReadLT      string
	DateReadLTE     string
	Grade           float64
	GradingCompany  GradingCompany
	IsRead          *bool
	IssueSeriesID   int
	ModifiedGT      string
	Page            int
	PurchaseDate    string
	PurchaseDateGT  string
	PurchaseDateGTE string
	PurchaseDateLT  string
	PurchaseDateLTE string
	PurchaseStore   string
	Rating          int
	StorageLocation string
}

func (o CollectionListOptions) values() url.Values {
	values := url.Values{}
	setStr := func(key, value string) {
		if value != "" {
			values.Set(key, value)
		}
	}
	if o.BookFormat != "" {
		values.Set("book_format", string(o.BookFormat))
	}
	setStr("date_read", o.DateRead)
	setStr("date_read_gt", o.DateReadGT)
	setStr("date_read_gte", o.DateReadGTE)
	setStr("date_read_lt", o.DateReadLT)
	setStr("date_read_lte", o.DateReadLTE)
	if o.Grade > 0 {
		values.Set("grade", strconv.FormatFloat(o.Grade, 'f', -1, 64))
	}
	if o.GradingCompany != "" {
		values.Set("grading_company", string(o.GradingCompany))
	}
	if o.IsRead != nil {
		values.Set("is_read", strconv.FormatBool(*o.IsRead))
	}
	if o.IssueSeriesID > 0 {
		values.Set("issue__series", strconv.Itoa(o.IssueSeriesID))
	}
	setStr("modified_gt", o.ModifiedGT)
	setStr("purchase_date", o.PurchaseDate)
	setStr("purchase_date_gt", o.PurchaseDateGT)
	setStr("purchase_date_gte", o.PurchaseDateGTE)
	setStr("purchase_date_lt", o.PurchaseDateLT)
	setStr("purchase_date_lte", o.PurchaseDateLTE)
	setStr("purchase_store", o.PurchaseStore)
	if o.Rating > 0 {
		values.Set("rating", strconv.Itoa(o.Rating))
	}
	setStr("storage_location", o.StorageLocation)
	return setPage(values, o.Page)
}

// MissingIssue is an issue from an owned series that isn't yet in the
// user's collection.
type MissingIssue struct {
	ID        int             `json:"id"`
	Series    IssueListSeries `json:"series"`
	Number    string          `json:"number"`
	CoverDate string          `json:"cover_date"`
	StoreDate string          `json:"store_date"`
}

// MissingSeries is a series where the user owns some, but not all, issues.
type MissingSeries struct {
	ID                   int            `json:"id"`
	Name                 string         `json:"name"`
	SortName             string         `json:"sort_name"`
	YearBegan            int            `json:"year_began"`
	YearEnd              int            `json:"year_end"`
	Publisher            BasicPublisher `json:"publisher"`
	SeriesType           SeriesTypeRef  `json:"series_type"`
	TotalIssues          int            `json:"total_issues"`
	OwnedIssues          int            `json:"owned_issues"`
	MissingCount         int            `json:"missing_count"`
	CompletionPercentage float64        `json:"completion_percentage"`
}

// CollectionStats summarizes the authenticated user's collection.
type CollectionStats struct {
	TotalItems    int    `json:"total_items"`
	TotalQuantity int    `json:"total_quantity"`
	TotalValue    string `json:"total_value"`
	ReadCount     int    `json:"read_count"`
	UnreadCount   int    `json:"unread_count"`
	ByFormat      []struct {
		BookFormat string `json:"book_format"`
		Count      int    `json:"count"`
	} `json:"by_format"`
}

// ScrobbleRequest marks an issue as read (or updates its read metadata).
// Metron auto-creates a collection item if one doesn't already exist.
type ScrobbleRequest struct {
	IssueID  int        `json:"issue_id"`
	DateRead *time.Time `json:"date_read,omitempty"`
	Rating   *int       `json:"rating,omitempty"`
}

// ScrobbleResponse is returned after a successful scrobble.
type ScrobbleResponse struct {
	ID       int             `json:"id"`
	Issue    CollectionIssue `json:"issue"`
	IsRead   bool            `json:"is_read"`
	DateRead string          `json:"date_read"`
	Rating   *int            `json:"rating"`
	Created  bool            `json:"created"`
	Modified string          `json:"modified"`
}

// CollectionService handles /api/collection/ endpoints.
type CollectionService struct{ client *Client }

// List returns one page of the authenticated user's collection items.
func (s *CollectionService) List(ctx context.Context, opts CollectionListOptions) (PagedResponse[CollectionList], error) {
	return listPage[CollectionList](ctx, s.client, "/collection/", opts.values())
}

// All walks every page of the authenticated user's collection items.
func (s *CollectionService) All(ctx context.Context, opts CollectionListOptions) ([]CollectionList, error) {
	return listAll[CollectionList](ctx, s.client, "/collection/", opts.values())
}

// Get returns a single collection item, which must belong to the
// authenticated user.
func (s *CollectionService) Get(ctx context.Context, id int) (*Collection, error) {
	var item Collection
	if err := s.client.get(ctx, "/collection/"+strconv.Itoa(id)+"/", nil, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateRating replaces the rating (1-5, or nil to clear) on collection
// item id. Read-tracking fields aren't editable here; use Scrobble.
func (s *CollectionService) UpdateRating(ctx context.Context, id int, rating *int) (*CollectionRatingUpdate, error) {
	var updated CollectionRatingUpdate
	body := CollectionRatingUpdate{Rating: rating}
	if err := s.client.put(ctx, "/collection/"+strconv.Itoa(id)+"/", body, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// PatchRating partially updates the rating on collection item id.
func (s *CollectionService) PatchRating(ctx context.Context, id int, rating *int) (*CollectionRatingUpdate, error) {
	var updated CollectionRatingUpdate
	body := CollectionRatingUpdate{Rating: rating}
	if err := s.client.patch(ctx, "/collection/"+strconv.Itoa(id)+"/", body, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// MissingIssues returns one page of the missing issues for seriesID, i.e.
// issues from that series the user doesn't yet own.
func (s *CollectionService) MissingIssues(ctx context.Context, seriesID int, page int) (PagedResponse[MissingIssue], error) {
	return listPage[MissingIssue](ctx, s.client, "/collection/missing_issues/"+strconv.Itoa(seriesID)+"/", setPage(nil, page))
}

// AllMissingIssues walks every page of the missing issues for seriesID.
func (s *CollectionService) AllMissingIssues(ctx context.Context, seriesID int) ([]MissingIssue, error) {
	return listAll[MissingIssue](ctx, s.client, "/collection/missing_issues/"+strconv.Itoa(seriesID)+"/", nil)
}

// MissingSeries returns one page of series where the user owns some, but
// not all, issues.
func (s *CollectionService) MissingSeries(ctx context.Context, page int) (PagedResponse[MissingSeries], error) {
	return listPage[MissingSeries](ctx, s.client, "/collection/missing_series/", setPage(nil, page))
}

// AllMissingSeries walks every page of series where the user owns some,
// but not all, issues.
func (s *CollectionService) AllMissingSeries(ctx context.Context) ([]MissingSeries, error) {
	return listAll[MissingSeries](ctx, s.client, "/collection/missing_series/", nil)
}

// Scrobble marks an issue as read, auto-creating a collection item if
// needed.
func (s *CollectionService) Scrobble(ctx context.Context, req ScrobbleRequest) (*ScrobbleResponse, error) {
	var resp ScrobbleResponse
	if err := s.client.post(ctx, "/collection/scrobble/", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Stats returns statistics about the authenticated user's collection.
func (s *CollectionService) Stats(ctx context.Context) (*CollectionStats, error) {
	var stats CollectionStats
	if err := s.client.get(ctx, "/collection/stats/", nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
