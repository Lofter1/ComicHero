package api

type Comic struct {
	ID            int  `json:"id"                      db:"id"              doc:"Local comic identifier." example:"42"`
	MetronIssueID *int `json:"metronIssueId,omitempty" db:"metron_issue_id" doc:"Linked Metron issue identifier, when this comic was imported or matched." example:"123456"`

	Title       string `json:"title"       db:"-"           doc:"Generated display title built from series, seriesYear, and issue." example:"Batman (2011) #6"`
	Series      string `json:"series"      db:"series"      doc:"Series name." example:"Batman"`
	SeriesYear  int    `json:"seriesYear"  db:"series_year" doc:"Series start year or volume year used in the generated title." minimum:"0" example:"2011"`
	Issue       string `json:"issue"       db:"issue"       doc:"Issue number." example:"6.LR"`
	Publisher   string `json:"publisher"   db:"publisher"   doc:"Publisher name." example:"DC Comics"`
	CoverDate   string `json:"coverDate"   db:"cover_date"  doc:"Cover date as provided by the source." example:"2012-04-01"`
	CoverImage  string `json:"coverImage"  db:"cover_image" doc:"Absolute URL for the cover image." format:"uri" example:"https://static.metron.cloud/media/issue/cover.jpg"`
	Description string `json:"description" db:"description" doc:"Issue synopsis or notes."`
	Read        bool   `json:"read"        db:"read"        doc:"Whether the comic has been read." example:"false"`
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
	ReadingOrders []ReadingOrder `json:"readingOrders" doc:"Reading orders that include this comic."`
	Arcs          []Arc          `json:"arcs"          doc:"Story arcs that include this comic."`
	Characters    []Character    `json:"characters"    doc:"Characters appearing in this comic."`
}

type ComicListInput struct {
	Query          string `query:"q"              doc:"Case-insensitive text search across generated title metadata, publisher, and description." example:"batman"`
	Series         string `query:"series"         doc:"Filter comics by partial series name." example:"Batman"`
	Publisher      string `query:"publisher"      doc:"Filter comics by partial publisher name." example:"DC"`
	Read           string `query:"read"           doc:"Filter comics by read status. Use true or false." enum:"true,false" example:"false"`
	ReadingOrderID int    `query:"readingOrderId" doc:"Filter comics to those included in a reading order." example:"7"`
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

type Character struct {
	ID                int      `json:"id"                         db:"id"                  doc:"Local character identifier." example:"12"`
	MetronCharacterID *int     `json:"metronCharacterId,omitempty" db:"metron_character_id" doc:"Linked Metron character identifier, when imported." example:"100"`
	Name              string   `json:"name"                       db:"name"                doc:"Character name." example:"Batman"`
	Description       string   `json:"description"                db:"description"         doc:"Character description from Metron."`
	Image             string   `json:"image"                      db:"image"               doc:"Character image URL from Metron." format:"uri"`
	Favorite          bool     `json:"favorite"                   db:"favorite"            doc:"Whether this character is marked as a favorite." example:"true"`
	Progress          float64  `json:"progress"                   db:"progress"            doc:"Fraction of appearances marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
	Aliases           []string `json:"aliases"                    db:"-"                   doc:"Known character aliases."`
	AppearanceCount   int      `json:"appearanceCount"            db:"appearance_count"    doc:"Number of local comics this character appears in." example:"25"`
}

type CharacterDetail struct {
	Character
	Comics []Comic `json:"comics" doc:"Local comics where this character appears."`
}

type CharacterListInput struct {
	Query    string `query:"q"        doc:"Case-insensitive text search across character names and aliases." example:"bat"`
	Favorite string `query:"favorite" doc:"Filter characters by favorite status. Use true or false." enum:"true,false" example:"true"`
	Limit    int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset   int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
}

type CharacterInput struct {
	ID int `path:"id" doc:"Local character identifier." example:"12"`
}

type UpdateCharacterFavoriteInput struct {
	ID   int `path:"id" doc:"Local character identifier." example:"12"`
	Body struct {
		Favorite bool `json:"favorite" doc:"New favorite status." example:"true"`
	}
}

type CharacterListOutput struct {
	PaginationHeaders
	Body []Character
}

type ComicSeries struct {
	ID             int  `json:"id"                       db:"id"               doc:"Local series identifier." example:"5"`
	MetronSeriesID *int `json:"metronSeriesId,omitempty" db:"metron_series_id" doc:"Linked Metron series identifier, when known." example:"405"`

	Name        string   `json:"name"        db:"name"        doc:"Series name." example:"Batman"`
	SeriesYear  int      `json:"seriesYear"  db:"series_year" doc:"Series start year or volume year used in generated comic titles." minimum:"0" example:"2011"`
	Favorite    bool     `json:"favorite"    db:"favorite"    doc:"Whether this series is marked as a favorite." example:"true"`
	Publisher   string   `json:"publisher"   db:"publisher"   doc:"Publisher name from Metron series metadata." example:"DC Comics"`
	Volume      int      `json:"volume"      db:"volume"      doc:"Metron series volume number." example:"2"`
	YearEnd     int      `json:"yearEnd"     db:"year_end"    doc:"Final publication year from Metron, when known." example:"2016"`
	IssueCount  int      `json:"issueCount"  db:"issue_count" doc:"Issue count reported by Metron." example:"52"`
	Description string   `json:"description" db:"description" doc:"Series description from Metron."`
	Progress    float64  `json:"progress"    db:"progress"    doc:"Fraction of entries marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
	EntryCount  int      `json:"entryCount"  db:"entry_count" doc:"Number of local comic entries in this series." example:"12"`
	ReadCount   int      `json:"readCount"   db:"read_count"  doc:"Number of local comic entries marked read." example:"6"`
	CoverImage  string   `json:"coverImage"  db:"cover_image" doc:"First available local or remote cover image for the series." format:"uri"`
	Publishers  []string `json:"publishers"  db:"-"           doc:"Publishers represented by local entries in this series."`
}

type ComicSeriesDetail struct {
	ComicSeries
	Comics []Comic `json:"comics" doc:"Local comics in this series."`
}

type ComicSeriesListInput struct {
	Query    string `query:"q"        doc:"Case-insensitive text search across series names, publishers, years, and issue numbers." example:"batman"`
	Favorite string `query:"favorite" doc:"Filter series by favorite status. Use true or false." enum:"true,false" example:"true"`
	Limit    int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset   int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
}

type ComicSeriesInput struct {
	ID int `path:"id" doc:"Local series identifier." example:"5"`
}

type UpdateComicSeriesFavoriteInput struct {
	ID   int `path:"id" doc:"Local series identifier." example:"5"`
	Body struct {
		Favorite bool `json:"favorite" doc:"New favorite status." example:"true"`
	}
}

type ComicSeriesListOutput struct {
	PaginationHeaders
	Body []ComicSeries
}

type ComicSeriesDetailOutput struct {
	Body ComicSeriesDetail
}

type CharacterDetailOutput struct {
	MetronRateLimitHeaders
	Body CharacterDetail
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
		Read bool `json:"read" doc:"New read status." example:"true"`
	}
}

type UpdateComicFromMetronInput struct {
	ID   int `path:"id" doc:"Local comic identifier." example:"42"`
	Body struct {
		MetronIssueID int  `json:"metronIssueId" minimum:"1" doc:"Metron issue identifier to copy metadata from." example:"123456"`
		Force         bool `json:"force,omitempty" doc:"Bypass Metron conditional requests and download fresh issue metadata." example:"false"`
	}
}

type ReadingOrder struct {
	ID                  int  `json:"id"                                  db:"id"                     doc:"Local reading-order identifier." example:"7"`
	MetronReadingListID *int `json:"metronReadingListId,omitempty"       db:"metron_reading_list_id" doc:"Linked Metron reading-list identifier, when imported." example:"9876"`

	Name        string  `json:"name"        db:"name"        doc:"Reading-order name." example:"Batman: Court of Owls"`
	Description string  `json:"description" db:"description" doc:"Reading-order description or notes."`
	Image       string  `json:"image"       db:"image"       doc:"Reading-list thumbnail image URL from Metron, when imported." format:"uri"`
	Favorite    bool    `json:"favorite"    db:"favorite"    doc:"Whether this reading order is marked as a favorite." example:"true"`
	Progress    float64 `json:"progress"    db:"progress"    doc:"Fraction of entries marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
}

type ReadingOrderPayload struct {
	Name        string `json:"name"        minLength:"1" doc:"Reading-order name." example:"Batman: Court of Owls"`
	Description string `json:"description" doc:"Reading-order description or notes."`
	Favorite    bool   `json:"favorite"    doc:"Whether this reading order is marked as a favorite." example:"true"`
}

type ReadingOrderComicPayload struct {
	ComicID int    `json:"comicId" doc:"Local comic identifier to add to the reading order." example:"42"`
	Comment string `json:"comment" doc:"Optional note for this specific reading-order entry." example:"Read after issue 5"`
	Tags    string `json:"tags"    doc:"Comma-separated tags for this reading-order entry." example:"Main Story"`
}

type ReadingOrderComic struct {
	Comic
	Comment string `json:"comment" db:"comment" doc:"Per-entry reading-order note." example:"Tie-in"`
	Tags    string `json:"tags"    db:"tags"    doc:"Comma-separated entry tags synced from Metron or added locally." example:"Main Story"`
}

type ReadingOrderDetail struct {
	ReadingOrder
	Comics             []ReadingOrderComic `json:"comics"             doc:"Comics in reading-order position order, including child reading-order comics."`
	ChildReadingOrders []ReadingOrder      `json:"childReadingOrders" doc:"Reading orders referenced by this reading order."`
}

type ReadingOrderListInput struct {
	Query    string `query:"q"        doc:"Case-insensitive text search across name and description." example:"batman"`
	Favorite string `query:"favorite" doc:"Filter reading orders by favorite status. Use true or false." enum:"true,false" example:"true"`
	ComicID  int    `query:"comicId"  doc:"Filter reading orders to those containing a comic." example:"42"`
	Limit    int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset   int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
}

type ReadingOrderInput struct {
	ID int `path:"id" doc:"Local reading-order identifier." example:"7"`
}

type ReadingOrderListOutput struct {
	PaginationHeaders
	Body []ReadingOrder
}

type Arc struct {
	ID          int  `json:"id"                    db:"id"            doc:"Local arc identifier." example:"7"`
	MetronArcID *int `json:"metronArcId,omitempty" db:"metron_arc_id" doc:"Linked Metron story-arc identifier, when imported." example:"9876"`

	Name        string  `json:"name"        db:"name"        doc:"Story arc name." example:"Batman: Zero Year"`
	Description string  `json:"description" db:"description" doc:"Arc description or notes."`
	Image       string  `json:"image"       db:"image"       doc:"Story arc image URL from Metron." format:"uri"`
	Favorite    bool    `json:"favorite"    db:"favorite"    doc:"Whether this arc is marked as a favorite." example:"true"`
	Progress    float64 `json:"progress"    db:"progress"    doc:"Fraction of entries marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
}

type ArcPayload struct {
	Name        string `json:"name"        minLength:"1" doc:"Story arc name." example:"Batman: Zero Year"`
	Description string `json:"description" doc:"Arc description or notes."`
	Favorite    bool   `json:"favorite"    doc:"Whether this arc is marked as a favorite." example:"true"`
}

type ArcComicPayload struct {
	ComicID int    `json:"comicId" doc:"Local comic identifier to add to the arc." example:"42"`
	Comment string `json:"comment" doc:"Optional note for this specific arc entry." example:"Tie-in"`
}

type ArcComic struct {
	Comic
	Comment string `json:"comment" db:"comment" doc:"Per-entry arc note." example:"Tie-in"`
}

type ArcDetail struct {
	Arc
	Comics []ArcComic `json:"comics" doc:"Comics in arc position order."`
}

type ArcListInput struct {
	Query    string `query:"q"        doc:"Case-insensitive text search across name and description." example:"batman"`
	Favorite string `query:"favorite" doc:"Filter arcs by favorite status. Use true or false." enum:"true,false" example:"true"`
	ComicID  int    `query:"comicId"  doc:"Filter arcs to those containing a comic." example:"42"`
	Limit    int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset   int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
}

type ArcInput struct {
	ID int `path:"id" doc:"Local arc identifier." example:"7"`
}

type ArcListOutput struct {
	PaginationHeaders
	Body []Arc
}

type ArcDetailOutput struct {
	MetronRateLimitHeaders
	Body ArcDetail
}

type CreateArcInput struct {
	Body ArcPayload
}

type CreateArcOutput struct {
	Body Arc
}

type UpdateArcInput struct {
	ID   int `path:"id" doc:"Local arc identifier." example:"7"`
	Body ArcPayload
}

type SetArcComicsInput struct {
	ID   int `path:"id" doc:"Local arc identifier." example:"7"`
	Body struct {
		ComicIDs []int             `json:"comicIds,omitempty" doc:"Comic IDs in arc order. Use comics to include comments." example:"[42,43]"`
		Comics   []ArcComicPayload `json:"comics,omitempty"   doc:"Comics in arc order with optional per-entry comments."`
	}
}

type ReadingOrderDetailOutput struct {
	MetronRateLimitHeaders
	Body ReadingOrderDetail
}

type CreateReadingOrderInput struct {
	Body ReadingOrderPayload
}

type CreateReadingOrderOutput struct {
	Body ReadingOrder
}

type UpdateReadingOrderInput struct {
	ID   int `path:"id" doc:"Local reading-order identifier." example:"7"`
	Body ReadingOrderPayload
}

type SetReadingOrderComicsInput struct {
	ID   int `path:"id" doc:"Local reading-order identifier." example:"7"`
	Body struct {
		ComicIDs        []int                      `json:"comicIds,omitempty"        doc:"Comic IDs in reading order. Use comics to include comments." example:"[42,43]"`
		Comics          []ReadingOrderComicPayload `json:"comics,omitempty"          doc:"Comics in reading order with optional per-entry comments."`
		ReadingOrderIDs []int                      `json:"readingOrderIds,omitempty" doc:"Child reading-order IDs referenced by this reading order." example:"[7,8]"`
	}
}
