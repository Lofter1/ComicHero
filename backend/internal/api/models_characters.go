package api

type Character struct {
	ID                int      `json:"id"                         db:"id"                  doc:"Local character identifier." example:"12"`
	MetronCharacterID *int     `json:"metronCharacterId,omitempty" db:"metron_character_id" doc:"Linked Metron character identifier, when imported." example:"100"`
	Name              string   `json:"name"                       db:"name"                doc:"Character name." example:"Batman"`
	Description       string   `json:"description"                db:"description"         doc:"Character description from Metron."`
	Image             string   `json:"image"                      db:"image"               doc:"Character image URL from Metron." format:"uri"`
	Favorite          bool     `json:"favorite"                   db:"favorite"            doc:"Whether this character is marked as a favorite." example:"true"`
	FavoriteCount     int      `json:"favoriteCount"              db:"favorite_count"      doc:"Number of users who favorited this character." example:"12"`
	StartedCount      int      `json:"startedCount"               db:"started_count"       doc:"Number of users currently reading this character." example:"8"`
	StartedAt         *string  `json:"startedAt,omitempty"        db:"started_at"          doc:"When the current user formally started reading this character."`
	Progress          float64  `json:"progress"                   db:"progress"            doc:"Fraction of appearances marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
	Aliases           []string `json:"aliases"                    db:"-"                   doc:"Known character aliases."`
	AppearanceCount   int      `json:"appearanceCount"            db:"appearance_count"    doc:"Number of local comics this character appears in." example:"25"`
}

type CharacterDetail struct {
	Character
	Comics []Comic `json:"comics" doc:"Local comics where this character appears."`
}

type CharacterListInput struct {
	Query     string `query:"q"        doc:"Case-insensitive text search across character names and aliases." example:"bat"`
	Favorite  string `query:"favorite" doc:"Filter characters by favorite status. Use true or false." enum:"true,false" example:"true"`
	Started   string `query:"started" doc:"Filter characters by current-user started status." enum:"true,false" example:"true"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,appearances,aliases,progress,favoriteCount,startedCount" example:"name"`
	Direction string `query:"direction" doc:"Sort direction." enum:"asc,desc" example:"asc"`
	Limit     int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset    int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
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

type CharacterDetailOutput struct {
	MetronRateLimitHeaders
	Body CharacterDetail
}
