package api

type Arc struct {
	ID          int  `json:"id"                    db:"id"            doc:"Local arc identifier." example:"7"`
	MetronArcID *int `json:"metronArcId,omitempty" db:"metron_arc_id" doc:"Linked Metron story-arc identifier, when imported." example:"9876"`

	Name          string  `json:"name"        db:"name"        doc:"Story arc name." example:"Batman: Zero Year"`
	Description   string  `json:"description" db:"description" doc:"Arc description or notes."`
	Image         string  `json:"image"       db:"image"       doc:"Story arc image URL from Metron." format:"uri"`
	Favorite      bool    `json:"favorite"    db:"favorite"    doc:"Whether this arc is marked as a favorite." example:"true"`
	FavoriteCount int     `json:"favoriteCount" db:"favorite_count" doc:"Number of users who favorited this arc." example:"12"`
	StartedCount  int     `json:"startedCount"  db:"started_count"  doc:"Number of users currently reading this arc." example:"8"`
	StartedAt     *string `json:"startedAt,omitempty" db:"started_at" doc:"When the current user formally started reading this arc."`
	Progress      float64 `json:"progress"    db:"progress"    doc:"Fraction of entries marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
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
	Query     string `query:"q"        doc:"Case-insensitive text search across name and description." example:"batman"`
	Favorite  string `query:"favorite" doc:"Filter arcs by favorite status. Use true or false." enum:"true,false" example:"true"`
	Started   string `query:"started" doc:"Filter arcs by current-user started status." enum:"true,false" example:"true"`
	ComicID   int    `query:"comicId"  doc:"Filter arcs to those containing a comic." example:"42"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,progress,favoriteCount,startedCount" example:"name"`
	Direction string `query:"direction" doc:"Sort direction." enum:"asc,desc" example:"asc"`
	Limit     int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset    int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
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
