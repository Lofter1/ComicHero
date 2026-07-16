package api

type Comic struct {
	ID            int  `json:"id"                      db:"id"              doc:"Local comic identifier." example:"42"`
	MetronIssueID *int `json:"metronIssueId,omitempty" db:"metron_issue_id" doc:"Linked Metron issue identifier, when this comic was imported or matched." example:"123456"`
	SeriesID      *int `json:"seriesId,omitempty"      db:"series_id"       doc:"Local series identifier for this comic's series, when available." example:"5"`

	Title          string `json:"title"       db:"-"           doc:"Generated display title built from series, seriesYear, and issue." example:"Batman (2011) #6"`
	Series         string `json:"series"      db:"series"      doc:"Series name." example:"Batman"`
	SeriesYear     int    `json:"seriesYear"  db:"series_year" doc:"Series start year or volume year used in the generated title." minimum:"0" example:"2011"`
	Issue          string `json:"issue"       db:"issue"       doc:"Issue number." example:"6.LR"`
	Publisher      string `json:"publisher"   db:"publisher"   doc:"Publisher name." example:"DC Comics"`
	CoverDate      string `json:"coverDate"   db:"cover_date"  doc:"Cover date as provided by the source." example:"2012-04-01"`
	CoverImage     string `json:"coverImage"  db:"cover_image" doc:"Absolute URL for the cover image." format:"uri" example:"https://static.metron.cloud/media/issue/cover.jpg"`
	Description    string `json:"description" db:"description" doc:"Issue synopsis or notes."`
	MetronSyncedAt string `json:"-" db:"metron_synced_at"`
	Read           bool   `json:"read"        db:"read"        doc:"Whether the comic has been read." example:"false"`
	Skipped        bool   `json:"skipped"     db:"skipped"     doc:"Whether the comic has been skipped." example:"false"`
}

type ComicPayload struct {
	Series      string `json:"series"     minLength:"1" doc:"Series name." example:"Batman"`
	SeriesYear  int    `json:"seriesYear" minimum:"0"   doc:"Series start year or volume year used in the generated title." example:"2011"`
	Issue       string `json:"issue"      doc:"Issue number." example:"6.LR"`
	Publisher   string `json:"publisher"  doc:"Publisher name." example:"DC Comics"`
	CoverDate   string `json:"coverDate"  doc:"Cover date as free text or ISO date." example:"2012-04-01"`
	CoverImage  string `json:"coverImage" doc:"Absolute URL for the cover image." format:"uri" example:"https://static.metron.cloud/media/issue/cover.jpg"`
	Description string `json:"description" doc:"Issue synopsis or notes."`
	Read        bool   `json:"read"       doc:"Whether the comic has been read." example:"false"`
}

type ComicDetail struct {
	Comic
	SeriesID      *int           `json:"seriesId,omitempty" doc:"Local series identifier for this comic's series, when available." example:"5"`
	ReadingOrders []ReadingOrder `json:"readingOrders" doc:"Reading orders that include this comic."`
	Arcs          []Arc          `json:"arcs"          doc:"Story arcs that include this comic."`
	Characters    []Character    `json:"characters"    doc:"Characters appearing in this comic."`
}

type ComicListInput struct {
	Query          string `query:"q"              doc:"Case-insensitive text search across generated title metadata, publisher, and description." example:"batman"`
	Series         string `query:"series"         doc:"Filter comics by partial series name." example:"Batman"`
	Publisher      string `query:"publisher"      doc:"Filter comics by partial publisher name." example:"DC"`
	Status         string `query:"status"         doc:"Comma-separated issue status filters: unread, read, skipped. Omit or use all for every status." example:"read,skipped"`
	Read           string `query:"read"           doc:"Filter comics by read status. Use true or false." enum:"true,false" example:"false"`
	Skipped        string `query:"skipped"        doc:"Filter comics by skipped status. Use true or false." enum:"true,false" example:"true"`
	ReadingOrderID int    `query:"readingOrderId" doc:"Filter comics to those included in a reading order." example:"7"`
	ArcID          int    `query:"arcId"          doc:"Filter comics to those included in an arc." example:"7"`
	CharacterID    int    `query:"characterId"    doc:"Filter comics to those featuring a character." example:"12"`
	SeriesID       int    `query:"seriesId"       doc:"Filter comics to a local series." example:"5"`
	Sort           string `query:"sort"           doc:"Sort field." enum:"series,title,date,publisher,read" example:"series"`
	Direction      string `query:"direction"      doc:"Sort direction." enum:"asc,desc" example:"asc"`
	Limit          int    `query:"limit"          doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset         int    `query:"offset"         doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
}

type ComicInput struct {
	ID int `path:"id" doc:"Local comic identifier." example:"42"`
}

type ComicListOutput struct {
	MetronRateLimitHeaders
	PaginationHeaders
	Body []Comic
}

type ComicDetailOutput struct {
	MetronRateLimitHeaders
	Body ComicDetail
}

type CreateComicInput struct {
	Body ComicPayload
}

type UpdateComicInput struct {
	ID   int `path:"id" doc:"Local comic identifier." example:"42"`
	Body ComicPayload
}

type UpdateComicReadInput struct {
	ID   int `path:"id" doc:"Local comic identifier." example:"42"`
	Body struct {
		Read    *bool `json:"read,omitempty" doc:"New read status." example:"true"`
		Skipped *bool `json:"skipped,omitempty" doc:"New skipped status." example:"false"`
	}
}

type UpdateComicFromMetronInput struct {
	ID   int `path:"id" doc:"Local comic identifier." example:"42"`
	Body struct {
		MetronIssueID int  `json:"metronIssueId" minimum:"1" doc:"Metron issue identifier to copy metadata from." example:"123456"`
		Force         bool `json:"force,omitempty" doc:"Bypass Metron conditional requests and download fresh issue metadata." example:"false"`
	}
}
