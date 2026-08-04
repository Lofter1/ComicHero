package metron

import (
	"context"
	"net/url"
	"strconv"
)

// CharacterList is the slim shape returned by the character list endpoint.
type CharacterList struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Modified string `json:"modified"`
}

// Character is the full character shape returned by the character detail
// endpoint.
type Character struct {
	ID          int            `json:"id"`
	Name        string         `json:"name"`
	Alias       []string       `json:"alias"`
	Desc        string         `json:"desc"`
	Image       string         `json:"image"`
	Creators    []CreatorList  `json:"creators"`
	Teams       []TeamList     `json:"teams"`
	Universes   []UniverseList `json:"universes"`
	CVID        int            `json:"cv_id"`
	GCDID       int            `json:"gcd_id"`
	ResourceURL string         `json:"resource_url"`
	Modified    string         `json:"modified"`
}

// CharacterListOptions filters GET /api/character/.
type CharacterListOptions struct {
	CVID       int
	GCDID      int
	ModifiedGT string
	Name       string
	Page       int
}

func (o CharacterListOptions) values() url.Values {
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

// CharacterService handles /api/character/ endpoints.
type CharacterService struct{ client *Client }

// List returns one page of characters matching opts.
func (s *CharacterService) List(ctx context.Context, opts CharacterListOptions) (PagedResponse[CharacterList], error) {
	return listPage[CharacterList](ctx, s.client, "/character/", opts.values())
}

// All walks every page of characters matching opts.
func (s *CharacterService) All(ctx context.Context, opts CharacterListOptions) ([]CharacterList, error) {
	return listAll[CharacterList](ctx, s.client, "/character/", opts.values())
}

// Get returns a single character by its Metron ID.
func (s *CharacterService) Get(ctx context.Context, id int) (*Character, error) {
	var character Character
	if err := s.client.get(ctx, "/character/"+strconv.Itoa(id)+"/", nil, &character); err != nil {
		return nil, err
	}
	return &character, nil
}

// IssueList returns one page of the issues character id appears in.
func (s *CharacterService) IssueList(ctx context.Context, id int, page int) (PagedResponse[IssueList], error) {
	return listPage[IssueList](ctx, s.client, "/character/"+strconv.Itoa(id)+"/issue_list/", setPage(nil, page))
}

// AllIssues walks every page of the issues character id appears in.
func (s *CharacterService) AllIssues(ctx context.Context, id int) ([]IssueList, error) {
	return listAll[IssueList](ctx, s.client, "/character/"+strconv.Itoa(id)+"/issue_list/", nil)
}
