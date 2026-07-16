package api

type ComicSeries struct {
	ID             int  `json:"id"                       db:"id"               doc:"Local series identifier." example:"5"`
	MetronSeriesID *int `json:"metronSeriesId,omitempty" db:"metron_series_id" doc:"Linked Metron series identifier, when known." example:"405"`

	Name          string   `json:"name"        db:"name"        doc:"Series name." example:"Batman"`
	SeriesYear    int      `json:"seriesYear"  db:"series_year" doc:"Series start year or volume year used in generated comic titles." minimum:"0" example:"2011"`
	Favorite      bool     `json:"favorite"    db:"favorite"    doc:"Whether this series is marked as a favorite." example:"true"`
	FavoriteCount int      `json:"favoriteCount" db:"favorite_count" doc:"Number of users who favorited this series." example:"12"`
	StartedCount  int      `json:"startedCount"  db:"started_count"  doc:"Number of users currently reading this series." example:"8"`
	StartedAt     *string  `json:"startedAt,omitempty" db:"started_at" doc:"When the current user formally started reading this series."`
	Publisher     string   `json:"publisher"   db:"publisher"   doc:"Publisher name from Metron series metadata." example:"DC Comics"`
	Volume        int      `json:"volume"      db:"volume"      doc:"Metron series volume number." example:"2"`
	YearEnd       int      `json:"yearEnd"     db:"year_end"    doc:"Final publication year from Metron, when known." example:"2016"`
	IssueCount    int      `json:"issueCount"  db:"issue_count" doc:"Issue count reported by Metron." example:"52"`
	Description   string   `json:"description" db:"description" doc:"Series description from Metron."`
	Progress      float64  `json:"progress"    db:"progress"    doc:"Fraction of entries marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
	EntryCount    int      `json:"entryCount"  db:"entry_count" doc:"Number of local comic entries in this series." example:"12"`
	ReadCount     int      `json:"readCount"   db:"read_count"  doc:"Number of local comic entries marked read." example:"6"`
	CoverImage    string   `json:"coverImage"  db:"cover_image" doc:"First available local or remote cover image for the series." format:"uri"`
	Publishers    []string `json:"publishers"  db:"-"           doc:"Publishers represented by local entries in this series."`
}

type ComicSeriesDetail struct {
	ComicSeries
	Comics []Comic `json:"comics" doc:"Local comics in this series."`
}

type ComicSeriesListInput struct {
	Query     string `query:"q"        doc:"Case-insensitive text search across series names, publishers, years, and issue numbers." example:"batman"`
	Favorite  string `query:"favorite" doc:"Filter series by favorite status. Use true or false." enum:"true,false" example:"true"`
	Started   string `query:"started" doc:"Filter series by current-user started status." enum:"true,false" example:"true"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,year,publisher,entries,progress,favoriteCount,startedCount" example:"name"`
	Direction string `query:"direction" doc:"Sort direction." enum:"asc,desc" example:"asc"`
	Limit     int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset    int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
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
