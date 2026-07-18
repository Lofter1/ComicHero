package api

type CharacterCollection struct {
	ID                int     `json:"id"                db:"id"                doc:"Local collection identifier." example:"4"`
	Name              string  `json:"name"              db:"name"              doc:"Private collection name." example:"Spider-Verse"`
	CreatedAt         string  `json:"createdAt"         db:"created_at"        doc:"When the collection was created."`
	StartedAt         *string `json:"startedAt,omitempty" db:"started_at"       doc:"When the owner started reading this collection."`
	CharacterCount    int     `json:"characterCount"    db:"character_count"   doc:"Number of characters in the collection." example:"5"`
	AppearanceCount   int     `json:"appearanceCount"   db:"appearance_count"  doc:"Number of distinct comics featuring collection characters." example:"72"`
	Progress          float64 `json:"progress"          db:"progress"          doc:"Fraction of distinct appearances marked read." minimum:"0" maximum:"1" example:"0.5"`
	ContainsCharacter bool    `json:"containsCharacter" db:"contains_character" doc:"Whether the character requested in the list filter is already a member."`
}

type CharacterCollectionDetail struct {
	CharacterCollection
	Characters []Character `json:"characters" doc:"Characters in this collection."`
	Comics     []Comic     `json:"comics"     doc:"Distinct local comics featuring any collection character, ordered by release date."`
}

type CharacterCollectionListInput struct {
	CharacterID int `query:"characterId" doc:"Optionally report membership for this character." example:"12"`
}

type CharacterCollectionInput struct {
	ID int `path:"id" doc:"Local collection identifier." example:"4"`
}

type CharacterCollectionMemberInput struct {
	ID          int `path:"id"          doc:"Local collection identifier." example:"4"`
	CharacterID int `path:"characterId" doc:"Local character identifier." example:"12"`
}

type CreateCharacterCollectionInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"120" doc:"Private collection name." example:"Spider-Verse"`
	}
}

type AddCharacterCollectionMemberInput struct {
	ID   int `path:"id" doc:"Local collection identifier." example:"4"`
	Body struct {
		CharacterID int `json:"characterId" minimum:"1" doc:"Character to add." example:"12"`
	}
}

type CharacterCollectionListOutput struct{ Body []CharacterCollection }
type CharacterCollectionOutput struct{ Body CharacterCollectionDetail }
