package metron

import (
	"context"
	"strconv"
)

// Priority is a wish-list item's priority (1 highest - 5 lowest).
type Priority int

// Currency is a supported price currency.
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyGBP Currency = "GBP"
)

// WishList is the authenticated user's wish list summary.
type WishList struct {
	ID        int    `json:"id"`
	ItemCount int    `json:"item_count"`
	ItemsURL  string `json:"items_url"`
	Modified  string `json:"modified"`
}

// WishListItemList is the slim shape returned by the wish list items
// endpoint.
type WishListItemList struct {
	ID           int             `json:"id"`
	Issue        CollectionIssue `json:"issue"`
	Status       string          `json:"status"`
	Priority     Priority        `json:"priority"`
	DesiredGrade *float64        `json:"desired_grade"`
	Modified     string          `json:"modified"`
}

// WishListItem is the full wish-list item shape returned after adding an
// item.
type WishListItem struct {
	ID               int             `json:"id"`
	Issue            CollectionIssue `json:"issue"`
	Status           string          `json:"status"`
	Priority         Priority        `json:"priority"`
	DesiredGrade     *float64        `json:"desired_grade"`
	MaxPrice         string          `json:"max_price"`
	MaxPriceCurrency string          `json:"max_price_currency"`
	Notes            string          `json:"notes"`
	AddedOn          string          `json:"added_on"`
	Modified         string          `json:"modified"`
}

// WishListAddItem is the request body for adding an issue to the
// authenticated user's wish list.
type WishListAddItem struct {
	IssueID          int      `json:"issue_id"`
	Priority         int      `json:"priority,omitempty"`
	DesiredGrade     string   `json:"desired_grade,omitempty"`
	MaxPrice         string   `json:"max_price,omitempty"`
	MaxPriceCurrency Currency `json:"max_price_currency,omitempty"`
	Notes            string   `json:"notes,omitempty"`
}

// AcquireWishListItem is the request body for marking a wish-list item as
// acquired, which creates a matching collection item.
type AcquireWishListItem struct {
	PurchaseDate          string   `json:"purchase_date,omitempty"`
	PurchasePrice         string   `json:"purchase_price,omitempty"`
	PurchasePriceCurrency Currency `json:"purchase_price_currency,omitempty"`
	PurchaseStore         string   `json:"purchase_store,omitempty"`
	Notes                 string   `json:"notes,omitempty"`
}

// WishListService handles /api/wish_list/ endpoints.
type WishListService struct{ client *Client }

// List returns one page of the authenticated user's wish lists.
func (s *WishListService) List(ctx context.Context, page int) (PagedResponse[WishList], error) {
	return listPage[WishList](ctx, s.client, "/wish_list/", setPage(nil, page))
}

// Get returns a single wish list by its Metron ID.
func (s *WishListService) Get(ctx context.Context, id int) (*WishList, error) {
	var list WishList
	if err := s.client.get(ctx, "/wish_list/"+strconv.Itoa(id)+"/", nil, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// Items returns one page of the authenticated user's wish list items.
func (s *WishListService) Items(ctx context.Context, page int) (PagedResponse[WishListItemList], error) {
	return listPage[WishListItemList](ctx, s.client, "/wish_list/items/", setPage(nil, page))
}

// AllItems walks every page of the authenticated user's wish list items.
func (s *WishListService) AllItems(ctx context.Context) ([]WishListItemList, error) {
	return listAll[WishListItemList](ctx, s.client, "/wish_list/items/", nil)
}

// AddItem adds an issue to the authenticated user's wish list.
func (s *WishListService) AddItem(ctx context.Context, item WishListAddItem) (*WishListItem, error) {
	var added WishListItem
	if err := s.client.post(ctx, "/wish_list/items/add/", item, &added); err != nil {
		return nil, err
	}
	return &added, nil
}

// RemoveItem removes itemID from the authenticated user's wish list.
func (s *WishListService) RemoveItem(ctx context.Context, itemID int) error {
	return s.client.delete(ctx, "/wish_list/items/"+strconv.Itoa(itemID)+"/remove/")
}

// AcquireItem marks wish-list item itemID as acquired, creating a matching
// collection item from acquisition.
func (s *WishListService) AcquireItem(ctx context.Context, itemID int, acquisition AcquireWishListItem) error {
	return s.client.post(ctx, "/wish_list/items/"+strconv.Itoa(itemID)+"/acquire/", acquisition, nil)
}
