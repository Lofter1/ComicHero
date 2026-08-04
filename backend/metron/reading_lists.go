package metron

import (
	"context"
	"net/url"
	"strconv"
)

// AttributionSource is the source a reading list's ordering was obtained
// from.
type AttributionSource string

const (
	AttributionSourceCBH   AttributionSource = "CBH"
	AttributionSourceCBRO  AttributionSource = "CBRO"
	AttributionSourceCBT   AttributionSource = "CBT"
	AttributionSourceCMRO  AttributionSource = "CMRO"
	AttributionSourceHTLC  AttributionSource = "HTLC"
	AttributionSourceLOCG  AttributionSource = "LOCG"
	AttributionSourceMG    AttributionSource = "MG"
	AttributionSourceOther AttributionSource = "OTHER"
)

// ReadingListType is the kind of subject a reading list is organized
// around.
type ReadingListType string

const (
	ReadingListTypeCharacters ReadingListType = "CHARACTERS"
	ReadingListTypeCreator    ReadingListType = "CREATOR"
	ReadingListTypeEvent      ReadingListType = "EVENT"
	ReadingListTypeMaster     ReadingListType = "MASTER"
	ReadingListTypeStory      ReadingListType = "STORY"
	ReadingListTypeTeams      ReadingListType = "TEAMS"
)

// ReadingListIssue is the slim issue shape embedded in reading list items.
type ReadingListIssue struct {
	ID        int             `json:"id"`
	Series    IssueListSeries `json:"series"`
	Number    string          `json:"number"`
	CoverDate string          `json:"cover_date"`
	StoreDate string          `json:"store_date"`
	CVID      int             `json:"cv_id"`
	GCDID     int             `json:"gcd_id"`
	Modified  string          `json:"modified"`
}

// ReadingListItem is one entry in a reading list, in reading order.
type ReadingListItem struct {
	ID        int              `json:"id"`
	Issue     ReadingListIssue `json:"issue"`
	Order     int              `json:"order"`
	IssueType string           `json:"issue_type"`
}

// ReadingListList is the slim shape returned by the reading list list
// endpoint.
type ReadingListList struct {
	ID                int               `json:"id"`
	Name              string            `json:"name"`
	Slug              string            `json:"slug"`
	User              User              `json:"user"`
	ListType          string            `json:"list_type"`
	IsPrivate         bool              `json:"is_private"`
	AttributionSource AttributionSource `json:"attribution_source"`
	AverageRating     float64           `json:"average_rating"`
	RatingCount       int               `json:"rating_count"`
	Modified          string            `json:"modified"`
}

// ReadingListNav is the minimal reading-list shape used for previous/next
// navigation links.
type ReadingListNav struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ReadingList is the full reading-list shape returned by the reading list
// detail endpoint.
type ReadingList struct {
	ID                int             `json:"id"`
	User              User            `json:"user"`
	Name              string          `json:"name"`
	Slug              string          `json:"slug"`
	Desc              string          `json:"desc"`
	Image             string          `json:"image"`
	ListType          string          `json:"list_type"`
	IsPrivate         bool            `json:"is_private"`
	AttributionSource string          `json:"attribution_source"`
	AttributionURL    string          `json:"attribution_url"`
	Previous          *ReadingListNav `json:"previous"`
	Next              *ReadingListNav `json:"next"`
	AverageRating     float64         `json:"average_rating"`
	RatingCount       int             `json:"rating_count"`
	ItemsURL          string          `json:"items_url"`
	ResourceURL       string          `json:"resource_url"`
	Modified          string          `json:"modified"`
}

// ReadingListListOptions filters GET /api/reading_list/.
type ReadingListListOptions struct {
	AttributionSource AttributionSource
	AverageRatingGTE  float64
	IsPrivate         *bool
	ListType          ReadingListType
	ModifiedGT        string
	Name              string
	Page              int
	Publisher         string
	UserID            int
	Username          string
}

func (o ReadingListListOptions) values() url.Values {
	values := url.Values{}
	if o.AttributionSource != "" {
		values.Set("attribution_source", string(o.AttributionSource))
	}
	if o.AverageRatingGTE > 0 {
		values.Set("average_rating__gte", strconv.FormatFloat(o.AverageRatingGTE, 'f', -1, 64))
	}
	if o.IsPrivate != nil {
		values.Set("is_private", strconv.FormatBool(*o.IsPrivate))
	}
	if o.ListType != "" {
		values.Set("list_type", string(o.ListType))
	}
	if o.ModifiedGT != "" {
		values.Set("modified_gt", o.ModifiedGT)
	}
	if o.Name != "" {
		values.Set("name", o.Name)
	}
	if o.Publisher != "" {
		values.Set("publisher", o.Publisher)
	}
	if o.UserID > 0 {
		values.Set("user", strconv.Itoa(o.UserID))
	}
	if o.Username != "" {
		values.Set("username", o.Username)
	}
	return setPage(values, o.Page)
}

// ReadingListService handles /api/reading_list/ endpoints.
type ReadingListService struct{ client *Client }

// List returns one page of reading lists visible to the authenticated
// user (their own lists plus public lists; admins additionally see
// Metron's own lists).
func (s *ReadingListService) List(ctx context.Context, opts ReadingListListOptions) (PagedResponse[ReadingListList], error) {
	return listPage[ReadingListList](ctx, s.client, "/reading_list/", opts.values())
}

// All walks every page of reading lists visible to the authenticated user.
func (s *ReadingListService) All(ctx context.Context, opts ReadingListListOptions) ([]ReadingListList, error) {
	return listAll[ReadingListList](ctx, s.client, "/reading_list/", opts.values())
}

// Get returns a single reading list by its Metron ID.
func (s *ReadingListService) Get(ctx context.Context, id int) (*ReadingList, error) {
	var list ReadingList
	if err := s.client.get(ctx, "/reading_list/"+strconv.Itoa(id)+"/", nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// Items returns one page of reading list id's items, in reading order.
func (s *ReadingListService) Items(ctx context.Context, id int, page int) (PagedResponse[ReadingListItem], error) {
	return listPage[ReadingListItem](ctx, s.client, "/reading_list/"+strconv.Itoa(id)+"/items/", setPage(nil, page))
}

// AllItems walks every page of reading list id's items, in reading order.
func (s *ReadingListService) AllItems(ctx context.Context, id int) ([]ReadingListItem, error) {
	return listAll[ReadingListItem](ctx, s.client, "/reading_list/"+strconv.Itoa(id)+"/items/", nil)
}
