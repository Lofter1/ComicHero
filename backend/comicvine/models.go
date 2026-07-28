package comicvine

import "time"

// Image represents an image in Comic Vine
type Image struct {
	IconURL        string `json:"icon_url"`
	MediumURL      string `json:"medium_url"`
	ScreenURL      string `json:"screen_url"`
	ScreenLargeURL string `json:"screen_large_url"`
	SmallURL       string `json:"small_url"`
	SuperURL       string `json:"super_url"`
	ThumbURL       string `json:"thumb_url"`
	TinyURL        string `json:"tiny_url"`
	OriginalURL    string `json:"original_url"`
	ImageTags      string `json:"image_tags"`
}

// Publisher represents a comic publisher
type Publisher struct {
	Aliases         string      `json:"aliases"`
	APIDetailURL    string      `json:"api_detail_url"`
	Characters      []Character `json:"characters"`
	DateAdded       time.Time   `json:"date_added"`
	DateLastUpdated time.Time   `json:"date_last_updated"`
	Deck            string      `json:"deck"`
	Description     string      `json:"description"`
	ID              int         `json:"id"`
	Image           Image       `json:"image"`
	LocationAddress string      `json:"location_address"`
	LocationCity    string      `json:"location_city"`
	LocationState   string      `json:"location_state"`
	Name            string      `json:"name"`
	SiteDetailURL   string      `json:"site_detail_url"`
	StoryArcs       []StoryArc  `json:"story_arcs"`
	Teams           []Team      `json:"teams"`
	Volumes         []Volume    `json:"volumes"`
}

// Character represents a comic character
type Character struct {
	Aliases                 string      `json:"aliases"`
	APIDetailURL            string      `json:"api_detail_url"`
	Birth                   interface{} `json:"birth"`
	CharacterEnemies        []Character `json:"character_enemies"`
	CharacterFriends        []Character `json:"character_friends"`
	CountOfIssueAppearances int         `json:"count_of_issue_appearances"`
	Creators                []Person    `json:"creators"`
	DateAdded               time.Time   `json:"date_added"`
	DateLastUpdated         time.Time   `json:"date_last_updated"`
	Deck                    string      `json:"deck"`
	Description             string      `json:"description"`
	FirstAppearedInIssue    Issue       `json:"first_appeared_in_issue"`
	Gender                  int         `json:"gender"`
	ID                      int         `json:"id"`
	Image                   Image       `json:"image"`
	IssuesDiedIn            []Issue     `json:"issues_died_in"`
	Movies                  []Movie     `json:"movies"`
	Name                    string      `json:"name"`
	Origin                  Origin      `json:"origin"`
	Publisher               Publisher   `json:"publisher"`
	Powers                  []Power     `json:"powers"`
	RealName                string      `json:"real_name"`
	SiteDetailURL           string      `json:"site_detail_url"`
	StoryArcs               []StoryArc  `json:"story_arcs"`
	Teams                   []Team      `json:"teams"`
	VolumeCredits           []Volume    `json:"volume_credits"`
}

// Concept represents a comic concept
type Concept struct {
	Aliases         string    `json:"aliases"`
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	Description     string    `json:"description"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Name            string    `json:"name"`
	SiteDetailURL   string    `json:"site_detail_url"`
	StartYear       string    `json:"start_year"`
}

// Episode represents a TV episode
type Episode struct {
	Aliases         string    `json:"aliases"`
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	Description     string    `json:"description"`
	EpisodeNumber   string    `json:"episode_number"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Name            string    `json:"name"`
	SeasonNumber    string    `json:"season_number"`
	Series          Series    `json:"series"`
	SiteDetailURL   string    `json:"site_detail_url"`
}

// Issue represents a comic issue
type Issue struct {
	Aliases                   string      `json:"aliases"`
	APIDetailURL              string      `json:"api_detail_url"`
	CharacterCredits          []Character `json:"character_credits"`
	CharacterDiedIn           []Character `json:"character_died_in"`
	ConceptCredits            []Concept   `json:"concept_credits"`
	CoverDate                 string      `json:"cover_date"`
	DateAdded                 time.Time   `json:"date_added"`
	DateLastUpdated           time.Time   `json:"date_last_updated"`
	Deck                      string      `json:"deck"`
	Description               string      `json:"description"`
	FirstAppearanceCharacters []Character `json:"first_appearance_characters"`
	FirstAppearanceConcepts   []Concept   `json:"first_appearance_concepts"`
	FirstAppearanceLocations  []Location  `json:"first_appearance_locations"`
	FirstAppearanceObjects    []Object    `json:"first_appearance_objects"`
	FirstAppearanceStoryArcs  []StoryArc  `json:"first_appearance_storyarcs"`
	FirstAppearanceTeams      []Team      `json:"first_appearance_teams"`
	HasStaffReview            bool        `json:"has_staff_review"`
	ID                        int         `json:"id"`
	Image                     Image       `json:"image"`
	IssueNumber               string      `json:"issue_number"`
	LocationCredits           []Location  `json:"location_credits"`
	Name                      string      `json:"name"`
	ObjectCredits             []Object    `json:"object_credits"`
	PersonCredits             []Person    `json:"person_credits"`
	SiteDetailURL             string      `json:"site_detail_url"`
	StoreDate                 string      `json:"store_date"`
	StoryArcCredits           []StoryArc  `json:"story_arc_credits"`
	TeamCredits               []Team      `json:"team_credits"`
	Volume                    *Volume     `json:"volume"`
}

// Location represents a comic location
type Location struct {
	Aliases         string    `json:"aliases"`
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	Description     string    `json:"description"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Name            string    `json:"name"`
	SiteDetailURL   string    `json:"site_detail_url"`
	StartYear       string    `json:"start_year"`
}

// Movie represents a movie
type Movie struct {
	Aliases         string      `json:"aliases"`
	APIDetailURL    string      `json:"api_detail_url"`
	Characters      []Character `json:"characters"`
	Concepts        []Concept   `json:"concepts"`
	DateAdded       time.Time   `json:"date_added"`
	DateLastUpdated time.Time   `json:"date_last_updated"`
	Deck            string      `json:"deck"`
	Description     string      `json:"description"`
	ID              int         `json:"id"`
	Image           Image       `json:"image"`
	Locations       []Location  `json:"locations"`
	Name            string      `json:"name"`
	Objects         []Object    `json:"objects"`
	People          []Person    `json:"people"`
	SiteDetailURL   string      `json:"site_detail_url"`
	Studios         []Studio    `json:"studios"`
	Teams           []Team      `json:"teams"`
}

// Object represents a comic object
type Object struct {
	Aliases         string    `json:"aliases"`
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	Description     string    `json:"description"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Name            string    `json:"name"`
	SiteDetailURL   string    `json:"site_detail_url"`
	StartYear       string    `json:"start_year"`
}

// Origin represents a character origin
type Origin struct {
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	SiteDetailURL   string    `json:"site_detail_url"`
}

// Person represents a person (creator)
type Person struct {
	Aliases           string      `json:"aliases"`
	APIDetailURL      string      `json:"api_detail_url"`
	Birth             string      `json:"birth"`
	Country           string      `json:"country"`
	CreatedCharacters []Character `json:"created_characters"`
	DateAdded         time.Time   `json:"date_added"`
	DateLastUpdated   time.Time   `json:"date_last_updated"`
	Death             interface{} `json:"death"`
	Deck              string      `json:"deck"`
	Description       string      `json:"description"`
	Email             string      `json:"email"`
	Gender            int         `json:"gender"`
	Hometown          string      `json:"hometown"`
	ID                int         `json:"id"`
	Image             Image       `json:"image"`
	Name              string      `json:"name"`
	SiteDetailURL     string      `json:"site_detail_url"`
	Website           string      `json:"website"`
}

// Power represents a character power
type Power struct {
	Aliases         string      `json:"aliases"`
	APIDetailURL    string      `json:"api_detail_url"`
	Characters      []Character `json:"characters"`
	DateAdded       time.Time   `json:"date_added"`
	DateLastUpdated time.Time   `json:"date_last_updated"`
	Description     string      `json:"description"`
	ID              int         `json:"id"`
	Name            string      `json:"name"`
	SiteDetailURL   string      `json:"site_detail_url"`
}

// Series represents a TV series
type Series struct {
	Aliases         string    `json:"aliases"`
	APIDetailURL    string    `json:"api_detail_url"`
	CountOfEpisodes int       `json:"count_of_episodes"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	Description     string    `json:"description"`
	Episodes        []Episode `json:"episodes"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Name            string    `json:"name"`
	Publisher       Publisher `json:"publisher"`
	SiteDetailURL   string    `json:"site_detail_url"`
	StartYear       string    `json:"start_year"`
}

// StoryArc represents a story arc
type StoryArc struct {
	Aliases         string    `json:"aliases"`
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	Description     string    `json:"description"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Name            string    `json:"name"`
	Publisher       Publisher `json:"publisher"`
	SiteDetailURL   string    `json:"site_detail_url"`
}

// Studio represents a movie studio
type Studio struct {
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	Description     string    `json:"description"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Name            string    `json:"name"`
	SiteDetailURL   string    `json:"site_detail_url"`
}

// Team represents a comic team
type Team struct {
	Aliases                 string      `json:"aliases"`
	APIDetailURL            string      `json:"api_detail_url"`
	CharacterEnemies        []Character `json:"character_enemies"`
	CharacterFriends        []Character `json:"character_friends"`
	Characters              []Character `json:"characters"`
	CountOfIssueAppearances int         `json:"count_of_issue_appearances"`
	DateAdded               time.Time   `json:"date_added"`
	DateLastUpdated         time.Time   `json:"date_last_updated"`
	Deck                    string      `json:"deck"`
	Description             string      `json:"description"`
	FirstAppearedInIssue    Issue       `json:"first_appeared_in_issue"`
	ID                      int         `json:"id"`
	Image                   Image       `json:"image"`
	IssuesDiedIn            []Issue     `json:"issues_died_in"`
	Movies                  []Movie     `json:"movies"`
	Name                    string      `json:"name"`
	Publisher               Publisher   `json:"publisher"`
	SiteDetailURL           string      `json:"site_detail_url"`
	StoryArcs               []StoryArc  `json:"story_arcs"`
	VolumeCredits           []Volume    `json:"volume_credits"`
}

// Volume represents a comic volume
type Volume struct {
	Aliases         string      `json:"aliases"`
	APIDetailURL    string      `json:"api_detail_url"`
	Characters      []Character `json:"characters"`
	Concepts        []Concept   `json:"concepts"`
	CountOfIssues   int         `json:"count_of_issues"`
	DateAdded       time.Time   `json:"date_added"`
	DateLastUpdated time.Time   `json:"date_last_updated"`
	Deck            string      `json:"deck"`
	Description     string      `json:"description"`
	FirstIssue      *Issue      `json:"first_issue"`
	ID              int         `json:"id"`
	Image           Image       `json:"image"`
	Issues          []Issue     `json:"issues"`
	LastIssue       *Issue      `json:"last_issue"`
	Locations       []Location  `json:"locations"`
	Name            string      `json:"name"`
	Objects         []Object    `json:"objects"`
	People          []Person    `json:"people"`
	Publisher       Publisher   `json:"publisher"`
	SiteDetailURL   string      `json:"site_detail_url"`
	StartYear       string      `json:"start_year"`
}

// Promo represents a promotional item
type Promo struct {
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Link            string    `json:"link"`
	Name            string    `json:"name"`
}

// Video represents a video
type Video struct {
	APIDetailURL    string    `json:"api_detail_url"`
	DateAdded       time.Time `json:"date_added"`
	DateLastUpdated time.Time `json:"date_last_updated"`
	Deck            string    `json:"deck"`
	ID              int       `json:"id"`
	Image           Image     `json:"image"`
	Link            string    `json:"link"`
	Name            string    `json:"name"`
	SiteDetailURL   string    `json:"site_detail_url"`
}

// Resource types for typed responses
type CharactersResponse struct {
	ListResponse
	Results []Character `json:"results"`
}

type ConceptsResponse struct {
	ListResponse
	Results []Concept `json:"results"`
}

type EpisodesResponse struct {
	ListResponse
	Results []Episode `json:"results"`
}

type IssuesResponse struct {
	ListResponse
	Results []Issue `json:"results"`
}

type LocationsResponse struct {
	ListResponse
	Results []Location `json:"results"`
}

type MoviesResponse struct {
	ListResponse
	Results []Movie `json:"results"`
}

type ObjectsResponse struct {
	ListResponse
	Results []Object `json:"results"`
}

type OriginsResponse struct {
	ListResponse
	Results []Origin `json:"results"`
}

type PowersResponse struct {
	ListResponse
	Results []Power `json:"results"`
}

type PublishersResponse struct {
	ListResponse
	Results []Publisher `json:"results"`
}

type SeriesResponse struct {
	ListResponse
	Results []Series `json:"results"`
}

type StoryArcsResponse struct {
	ListResponse
	Results []StoryArc `json:"results"`
}

type TeamsResponse struct {
	ListResponse
	Results []Team `json:"results"`
}

type VolumesResponse struct {
	ListResponse
	Results []Volume `json:"results"`
}

type PromosResponse struct {
	ListResponse
	Results []Promo `json:"results"`
}

type VideosResponse struct {
	ListResponse
	Results []Video `json:"results"`
}

// Detailed response types for single resources
type CharacterDetailedResponse struct {
	Error      string    `json:"error"`
	StatusCode int       `json:"status_code"`
	Version    string    `json:"version"`
	Results    Character `json:"results"`
}

type ConceptDetailedResponse struct {
	Error      string  `json:"error"`
	StatusCode int     `json:"status_code"`
	Version    string  `json:"version"`
	Results    Concept `json:"results"`
}

type EpisodeDetailedResponse struct {
	Error      string  `json:"error"`
	StatusCode int     `json:"status_code"`
	Version    string  `json:"version"`
	Results    Episode `json:"results"`
}

type IssueDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Issue  `json:"results"`
}

type LocationDetailedResponse struct {
	Error      string   `json:"error"`
	StatusCode int      `json:"status_code"`
	Version    string   `json:"version"`
	Results    Location `json:"results"`
}

type MovieDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Movie  `json:"results"`
}

type ObjectDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Object `json:"results"`
}

type OriginDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Origin `json:"results"`
}

type PowerDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Power  `json:"results"`
}

type PublisherDetailedResponse struct {
	Error      string    `json:"error"`
	StatusCode int       `json:"status_code"`
	Version    string    `json:"version"`
	Results    Publisher `json:"results"`
}

type SeriesDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Series `json:"results"`
}

type StoryArcDetailedResponse struct {
	Error      string   `json:"error"`
	StatusCode int      `json:"status_code"`
	Version    string   `json:"version"`
	Results    StoryArc `json:"results"`
}

type TeamDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Team   `json:"results"`
}

type VolumeDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Volume `json:"results"`
}

type PromoDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Promo  `json:"results"`
}

type VideoDetailedResponse struct {
	Error      string `json:"error"`
	StatusCode int    `json:"status_code"`
	Version    string `json:"version"`
	Results    Video  `json:"results"`
}
