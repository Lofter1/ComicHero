package metron

import (
	"fmt"
	"time"
)

type Issue struct {
	ID           int               `json:"id"          doc:"Metron issue identifier." example:"123456"`
	ComicVineID  int               `json:"comicVineId" doc:"Comic Vine issue identifier returned by Metron, when known." example:"987654"`
	Title        string            `json:"title"       doc:"Issue title." example:"The Court of Owls"`
	StoryNames   []string          `json:"storyNames"  doc:"Story titles returned by Metron."`
	SeriesID     int               `json:"seriesId"    doc:"Metron series identifier for this issue." example:"405"`
	Series       string            `json:"series"      doc:"Series name." example:"Batman"`
	SeriesYear   int               `json:"seriesYear"  doc:"Series start year or volume year used by ComicHero generated titles." example:"2011"`
	SeriesVolume int               `json:"seriesVolume" doc:"Metron series volume number." example:"2"`
	Issue        string            `json:"issue"       doc:"Issue number as provided by Metron." example:"6.LR"`
	Number       string            `json:"number"      doc:"Original Metron issue number." example:"6"`
	Publisher    string            `json:"publisher"   doc:"Publisher name." example:"DC Comics"`
	CoverDate    string            `json:"coverDate"   doc:"Cover date as provided by Metron." example:"2012-04-01"`
	StoreDate    string            `json:"storeDate"   doc:"Store date as provided by Metron." example:"2012-04-01"`
	CoverImage   string            `json:"coverImage"  doc:"Absolute Metron cover-image URL." format:"uri" example:"https://static.metron.cloud/media/issue/cover.jpg"`
	Description  string            `json:"description" doc:"Metron issue synopsis."`
	Modified     string            `json:"modified"    doc:"Metron modified timestamp."`
	Tags         []string          `json:"tags"        doc:"Reading-list item tags/classifications from Metron."`
	Arcs         []MetronArc       `json:"arcs"        doc:"Story arcs attached to the issue, when returned by Metron."`
	Characters   []MetronCharacter `json:"characters"  doc:"Characters appearing in the issue, when returned by Metron."`
}

type MetronCharacter struct {
	ID          int      `json:"id"          doc:"Metron character identifier." example:"100"`
	Name        string   `json:"name"        doc:"Character name." example:"Batman"`
	Aliases     []string `json:"aliases"     doc:"Known character aliases from Metron."`
	Description string   `json:"description" doc:"Character description from Metron."`
	Image       string   `json:"image"       doc:"Character image URL from Metron." format:"uri"`
}

type ReadingList struct {
	ID                int        `json:"id"                doc:"Metron reading-list identifier." example:"9876"`
	Name              string     `json:"name"              doc:"Metron reading-list name." example:"Batman: Court of Owls"`
	Slug              string     `json:"slug,omitempty"    doc:"Metron reading-list slug."`
	User              MetronUser `json:"user,omitempty"    doc:"Metron user who owns the reading list."`
	ListType          string     `json:"listType"          doc:"Metron reading-list type."`
	IsPrivate         bool       `json:"isPrivate"         doc:"Whether the Metron reading list is private."`
	AttributionSource string     `json:"attributionSource" doc:"Reading-list attribution source."`
	AttributionURL    string     `json:"attributionUrl"    doc:"Reading-list attribution URL." format:"uri"`
	AverageRating     float64    `json:"averageRating"     doc:"Average Metron user rating."`
	RatingCount       int        `json:"ratingCount"       doc:"Number of Metron ratings."`
	Modified          string     `json:"modified"          doc:"Metron modified timestamp."`
	Image             string     `json:"image"             doc:"Metron reading-list image URL." format:"uri"`
	ItemsURL          string     `json:"itemsUrl"          doc:"Metron reading-list items URL." format:"uri"`
	ResourceURL       string     `json:"resourceUrl"       doc:"Metron reading-list resource URL." format:"uri"`
	Description       string     `json:"description"       doc:"Metron reading-list description."`
	Issues            []Issue    `json:"issues"            doc:"Issues included in the reading list, when requested from a detail endpoint."`
}

type MetronUser struct {
	ID       int    `json:"id"       doc:"Metron user identifier." example:"42"`
	Username string `json:"username" doc:"Metron username." example:"reader"`
}

type MetronArc struct {
	ID          int     `json:"id"          doc:"Metron story-arc identifier." example:"9876"`
	Name        string  `json:"name"        doc:"Metron story-arc name." example:"Batman: Zero Year"`
	Description string  `json:"description" doc:"Metron story-arc description."`
	Image       string  `json:"image"       doc:"Metron story-arc image URL." format:"uri"`
	Modified    string  `json:"modified"    doc:"Metron modified timestamp."`
	Issues      []Issue `json:"issues"      doc:"Issues included in the story arc, when requested from a detail endpoint."`
}

type Series struct {
	ID          int    `json:"id"                doc:"Metron series identifier." example:"405"`
	Name        string `json:"name"              doc:"Series name." example:"Batman"`
	Publisher   string `json:"publisher"         doc:"Publisher name." example:"DC Comics"`
	Volume      int    `json:"volume"            doc:"Series volume number." example:"2"`
	YearBegan   int    `json:"yearBegan"         doc:"First publication year." example:"2011"`
	YearEnd     int    `json:"yearEnd,omitempty" doc:"Final publication year, when the series has ended." example:"2016"`
	IssueCount  int    `json:"issueCount"        doc:"Number of issues reported by Metron." example:"52"`
	Description string `json:"description"       doc:"Metron series description."`
}

type RateLimit struct {
	BurstLimit         int   `json:"burstLimit,omitempty"         doc:"Metron burst-rate request limit."`
	BurstRemaining     int   `json:"burstRemaining,omitempty"     doc:"Remaining Metron burst-rate requests."`
	BurstReset         int64 `json:"burstReset,omitempty"         doc:"Unix timestamp when the burst-rate window resets."`
	SustainedLimit     int   `json:"sustainedLimit,omitempty"     doc:"Metron sustained-rate request limit."`
	SustainedRemaining int   `json:"sustainedRemaining,omitempty" doc:"Remaining Metron sustained-rate requests."`
	SustainedReset     int64 `json:"sustainedReset,omitempty"     doc:"Unix timestamp when the sustained-rate window resets."`
}

type ConditionalRequest struct {
	LastModified string
	Force        bool
}

type FetchInfo struct {
	LastModified string
	NotModified  bool
}

type RequestLogEntry struct {
	StartedAt      string `json:"startedAt"`
	Method         string `json:"method"`
	URL            string `json:"url"`
	Path           string `json:"path"`
	Query          string `json:"query"`
	Status         int    `json:"status"`
	DurationMillis int64  `json:"durationMillis"`
	Conditional    bool   `json:"conditional"`
	Error          string `json:"error,omitempty"`
}

type RateLimitError struct {
	Status    string
	Body      string
	RateLimit RateLimit
}

func (e *RateLimitError) Error() string {
	reset := e.RateLimit.NextReset()
	if reset == 0 {
		return "metron rate limit reached"
	}
	return fmt.Sprintf("metron rate limit reached; try again after %s", time.Unix(reset, 0).Format(time.RFC3339))
}
