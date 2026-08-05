package comicvine

import (
	"context"
	"net/url"
	"strconv"
)

// ResourceType represents the type of Comic Vine resource
type ResourceType string

const (
	ResourceCharacter  ResourceType = "character"
	ResourceConcept    ResourceType = "concept"
	ResourceEpisode    ResourceType = "episode"
	ResourceIssue      ResourceType = "issue"
	ResourceLocation   ResourceType = "location"
	ResourceMovie      ResourceType = "movie"
	ResourceObject     ResourceType = "object"
	ResourceOrigin     ResourceType = "origin"
	ResourcePower      ResourceType = "power"
	ResourcePublisher  ResourceType = "publisher"
	ResourceSeries     ResourceType = "series"
	ResourceStoryArc   ResourceType = "story_arc"
	ResourceTeam       ResourceType = "team"
	ResourceVolume     ResourceType = "volume"
	ResourcePromo      ResourceType = "promo"
	ResourceVideo      ResourceType = "video"
)

// BaseService provides common functionality for all resource services
type BaseService struct {
	client       *Client
	resourceType ResourceType
}

// ListOptions contains options for listing resources
type ListOptions struct {
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
	Sort      string   `json:"sort,omitempty"`
	Filter    string   `json:"filter,omitempty"`
	Fields    []string `json:"field_list,omitempty"`
}

// SearchOptions contains options for searching resources
type SearchOptions struct {
	Limit     int      `json:"limit,omitempty"`
	Page      int      `json:"page,omitempty"`
	Resources []string `json:"resources,omitempty"`
	Fields    []string `json:"field_list,omitempty"`
}

// ListResponse represents a paginated list response
type ListResponse struct {
	Error           string      `json:"error"`
	Limit           int         `json:"limit"`
	Offset          int         `json:"offset"`
	NumberOfResults int         `json:"number_of_total_results"`
	StatusCode      int         `json:"status_code"`
	Version         string      `json:"version"`
}

// idPrefixes maps each Comic Vine resource type to the numeric prefix Comic Vine
// requires in front of an object's numeric ID when fetching it directly (e.g.
// "4050-12345" for a volume, "4000-12345" for an issue). These prefixes are fixed
// and documented by Comic Vine itself (see the resource list at
// https://comicvine.gamespot.com/api/documentation, or /api/types/ with an API key)
// and do not vary by request.
var idPrefixes = map[ResourceType]string{
	ResourceCharacter: "4005",
	ResourceConcept:   "4015",
	ResourceEpisode:   "4070",
	ResourceIssue:     "4000",
	ResourceLocation:  "4020",
	ResourceMovie:     "4025",
	ResourceObject:    "4055",
	ResourceOrigin:    "4065",
	ResourcePower:     "4035",
	ResourcePromo:     "4090",
	ResourcePublisher: "4010",
	ResourceSeries:    "4075",
	ResourceStoryArc:  "4045",
	ResourceTeam:      "4060",
	ResourceVolume:    "4050",
	ResourceVideo:     "4085",
}

// idPrefixFor returns the Comic Vine ID prefix for a resource type, falling back
// to the issue prefix ("4000") for any resource type not in idPrefixes.
func idPrefixFor(resourceType ResourceType) string {
	if prefix, ok := idPrefixes[resourceType]; ok {
		return prefix
	}
	return idPrefixes[ResourceIssue]
}

// GetByID fetches a single resource by its ID
func (s *BaseService) GetByID(ctx context.Context, id int, fields []string, v interface{}) error {
	path := "/" + string(s.resourceType) + "/" + idPrefixFor(s.resourceType) + "-" + strconv.Itoa(id) + "/"
	params := url.Values{}
	
	if len(fields) > 0 {
		addFields(params, fields)
	}

	return s.client.getJSON(ctx, path, params, v)
}

// List fetches a list of resources
func (s *BaseService) List(ctx context.Context, opts *ListOptions, v interface{}) error {
	path := "/" + string(s.resourceType) + "s/"
	params := url.Values{}

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			params.Set("offset", strconv.Itoa(opts.Offset))
		}
		if opts.Sort != "" {
			params.Set("sort", opts.Sort)
		}
		if opts.Filter != "" {
			params.Set("filter", opts.Filter)
		}
		if len(opts.Fields) > 0 {
			addFields(params, opts.Fields)
		}
	}

	return s.client.getJSON(ctx, path, params, v)
}

// Search searches for resources by name
func (s *BaseService) Search(ctx context.Context, query string, opts *SearchOptions, v interface{}) error {
	path := "/search/"
	params := url.Values{}
	params.Set("query", query)
	params.Set("resources", string(s.resourceType))

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if len(opts.Fields) > 0 {
			addFields(params, opts.Fields)
		}
	}

	return s.client.getJSON(ctx, path, params, v)
}

// Get executes a custom query against the resource
func (s *BaseService) Get(ctx context.Context, query *Query, v interface{}) error {
	path := "/" + string(s.resourceType) + "s/"
	params := query.ToParams()
	return s.client.getJSON(ctx, path, params, v)
}

// CharacterService handles character-related API endpoints
type CharacterService struct {
	*BaseService
}

// CharacterResponse represents the API response for characters
type CharacterResponse struct {
	ListResponse
	Results []Character `json:"results"`
}

// SingleCharacterResponse represents the API response for a single character
type SingleCharacterResponse struct {
	Error      string    `json:"error"`
	StatusCode int       `json:"status_code"`
	Version    string    `json:"version"`
	Results    Character `json:"results"`
}

// GetByID fetches a single character by ID
func (s *CharacterService) GetByID(ctx context.Context, id int, fields []string) (*Character, error) {
	var resp SingleCharacterResponse
	if err := s.BaseService.GetByID(ctx, id, fields, &resp); err != nil {
		return nil, err
	}
	return &resp.Results, nil
}

// List returns a list of characters
func (s *CharacterService) List(ctx context.Context, opts *ListOptions) (*CharacterResponse, error) {
	var resp CharacterResponse
	if err := s.BaseService.List(ctx, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Search searches for characters by name
func (s *CharacterService) Search(ctx context.Context, query string, opts *SearchOptions) (*CharacterResponse, error) {
	var resp CharacterResponse
	if err := s.BaseService.Search(ctx, query, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get executes a custom query for characters
func (s *CharacterService) Get(ctx context.Context, query *Query) (*CharacterResponse, error) {
	var resp CharacterResponse
	if err := s.BaseService.Get(ctx, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// IssueService handles issue-related API endpoints
type IssueService struct {
	*BaseService
}

// IssueResponse represents the API response for issues
type IssueResponse struct {
	ListResponse
	Results []Issue `json:"results"`
}

// SingleIssueResponse represents the API response for a single issue
type SingleIssueResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Issue  `json:"results"`
}

// GetByID fetches a single issue by ID
func (s *IssueService) GetByID(ctx context.Context, id int, fields []string) (*Issue, error) {
	var resp SingleIssueResponse
	if err := s.BaseService.GetByID(ctx, id, fields, &resp); err != nil {
		return nil, err
	}
	return &resp.Results, nil
}

// List returns a list of issues
func (s *IssueService) List(ctx context.Context, opts *ListOptions) (*IssueResponse, error) {
	var resp IssueResponse
	if err := s.BaseService.List(ctx, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Search searches for issues
func (s *IssueService) Search(ctx context.Context, query string, opts *SearchOptions) (*IssueResponse, error) {
	var resp IssueResponse
	if err := s.BaseService.Search(ctx, query, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get executes a custom query for issues
func (s *IssueService) Get(ctx context.Context, query *Query) (*IssueResponse, error) {
	var resp IssueResponse
	if err := s.BaseService.Get(ctx, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// VolumeService handles volume-related API endpoints
type VolumeService struct {
	*BaseService
}

// VolumeResponse represents the API response for volumes
type VolumeResponse struct {
	ListResponse
	Results []Volume `json:"results"`
}

// SingleVolumeResponse represents the API response for a single volume
type SingleVolumeResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Volume `json:"results"`
}

// GetByID fetches a single volume by ID
func (s *VolumeService) GetByID(ctx context.Context, id int, fields []string) (*Volume, error) {
	var resp SingleVolumeResponse
	if err := s.BaseService.GetByID(ctx, id, fields, &resp); err != nil {
		return nil, err
	}
	return &resp.Results, nil
}

// List returns a list of volumes
func (s *VolumeService) List(ctx context.Context, opts *ListOptions) (*VolumeResponse, error) {
	var resp VolumeResponse
	if err := s.BaseService.List(ctx, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Search searches for volumes
func (s *VolumeService) Search(ctx context.Context, query string, opts *SearchOptions) (*VolumeResponse, error) {
	var resp VolumeResponse
	if err := s.BaseService.Search(ctx, query, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get executes a custom query for volumes
func (s *VolumeService) Get(ctx context.Context, query *Query) (*VolumeResponse, error) {
	var resp VolumeResponse
	if err := s.BaseService.Get(ctx, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PublisherService handles publisher-related API endpoints
type PublisherService struct {
	*BaseService
}

// PublisherResponse represents the API response for publishers
type PublisherResponse struct {
	ListResponse
	Results []Publisher `json:"results"`
}

// SinglePublisherResponse represents the API response for a single publisher
type SinglePublisherResponse struct {
	Error      string    `json:"error"`
	StatusCode int       `json:"status_code"`
	Version    string    `json:"version"`
	Results    Publisher `json:"results"`
}

// GetByID fetches a single publisher by ID
func (s *PublisherService) GetByID(ctx context.Context, id int, fields []string) (*Publisher, error) {
	var resp SinglePublisherResponse
	if err := s.BaseService.GetByID(ctx, id, fields, &resp); err != nil {
		return nil, err
	}
	return &resp.Results, nil
}

// List returns a list of publishers
func (s *PublisherService) List(ctx context.Context, opts *ListOptions) (*PublisherResponse, error) {
	var resp PublisherResponse
	if err := s.BaseService.List(ctx, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Search searches for publishers
func (s *PublisherService) Search(ctx context.Context, query string, opts *SearchOptions) (*PublisherResponse, error) {
	var resp PublisherResponse
	if err := s.BaseService.Search(ctx, query, opts, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Get executes a custom query for publishers
func (s *PublisherService) Get(ctx context.Context, query *Query) (*PublisherResponse, error) {
	var resp PublisherResponse
	if err := s.BaseService.Get(ctx, query, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Add remaining services for brevity. They all follow the same pattern.
type ConceptService struct{ *BaseService }
type EpisodeService struct{ *BaseService }
type LocationService struct{ *BaseService }
type MovieService struct{ *BaseService }
type ObjectService struct{ *BaseService }
type OriginService struct{ *BaseService }
type PowerService struct{ *BaseService }
type SeriesService struct{ *BaseService }
type StoryArcService struct{ *BaseService }
type TeamService struct{ *BaseService }
type PromoService struct{ *BaseService }
type VideoService struct{ *BaseService }

// SearchService handles global search across all resources
type SearchService struct {
	client *Client
}

// GlobalSearchResponse represents the API response for global search
type GlobalSearchResponse struct {
	ListResponse
	Results []SearchResult `json:"results"`
}

// Search performs a global search across multiple resource types
func (s *SearchService) Search(ctx context.Context, query string, resources []ResourceType, opts *SearchOptions) (*GlobalSearchResponse, error) {
	path := "/search/"
	params := url.Values{}
	params.Set("query", query)

	if len(resources) > 0 {
		resourceStrs := make([]string, len(resources))
		for i, r := range resources {
			resourceStrs[i] = string(r)
		}
		params["resources"] = resourceStrs
	}

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if len(opts.Fields) > 0 {
			addFields(params, opts.Fields)
		}
	}

	var resp GlobalSearchResponse
	if err := s.client.getJSON(ctx, path, params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SearchResult represents a search result
type SearchResult struct {
	Aliases           string      `json:"aliases"`
	APIDetailURL      string      `json:"api_detail_url"`
	DateAdded         string      `json:"date_added"`
	DateLastUpdated   string      `json:"date_last_updated"`
	Deck              string      `json:"deck"`
	Description       string      `json:"description"`
	ID                int         `json:"id"`
	Image             Image       `json:"image"`
	Name              string      `json:"name"`
	ResourceType      string      `json:"resource_type"`
	SiteDetailURL     string      `json:"site_detail_url"`
}

// Helper function to add field list to params
func addFields(params url.Values, fields []string) {
	fieldList := ""
	for i, field := range fields {
		if i > 0 {
			fieldList += ","
		}
		fieldList += field
	}
	params.Set("field_list", fieldList)
}

// Helper function to create BaseService for a resource type
func newBaseService(client *Client, resourceType ResourceType) *BaseService {
	return &BaseService{
		client:       client,
		resourceType: resourceType,
	}
}

// Initialize services with BaseService
func init() {
	// This is handled in the NewClient function instead
}
