package api

type ReadingOrder struct {
	ID                  int  `json:"id"                                  db:"id"                     doc:"Local reading-order identifier." example:"7"`
	MetronReadingListID *int `json:"metronReadingListId,omitempty"       db:"metron_reading_list_id" doc:"Linked Metron reading-list identifier, when imported." example:"9876"`
	AuthorUserID        *int `json:"authorUserId,omitempty"              db:"author_user_id"         doc:"User who created or owns this reading order." example:"1"`

	Name          string   `json:"name"        db:"name"        doc:"Reading-order name." example:"Batman: Court of Owls"`
	Description   string   `json:"description" db:"description" doc:"Reading-order description or notes."`
	Image         string   `json:"image"       db:"image"       doc:"Reading-list thumbnail image URL from Metron, when imported." format:"uri"`
	IsPublic      bool     `json:"isPublic"    db:"is_public"   doc:"Whether users other than the creator may view this reading order." example:"true"`
	Favorite      bool     `json:"favorite"    db:"favorite"    doc:"Whether this reading order is marked as a favorite." example:"true"`
	FavoriteCount int      `json:"favoriteCount" db:"favorite_count" doc:"Number of users who favorited this reading order." example:"12"`
	StartedCount  int      `json:"startedCount"  db:"started_count"  doc:"Number of users currently reading this reading order." example:"8"`
	Rating        float64  `json:"rating"      db:"rating"      doc:"Average user rating from 1 to 5, or 0 when unrated." minimum:"0" maximum:"5" example:"4.5"`
	RatingCount   int      `json:"ratingCount" db:"rating_count" doc:"Number of user ratings represented by this score." example:"23"`
	MyRating      *float64 `json:"myRating,omitempty" db:"my_rating" doc:"Current user's rating for this reading order, when rated." minimum:"1" maximum:"5" example:"4"`
	StartedAt     *string  `json:"startedAt,omitempty" db:"started_at" doc:"When the current user formally started this reading order." example:"2026-07-10T12:30:00Z"`
	Progress      float64  `json:"progress"    db:"progress"    doc:"Fraction of entries marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
	AuthorName    string   `json:"authorName"  db:"author_name" doc:"Display name of the reading-order author." example:"Default"`
	CanEdit       bool     `json:"canEdit"     db:"can_edit"    doc:"Whether the current user may edit this reading order." example:"true"`
}

type ReadingOrderPayload struct {
	Name           string `json:"name"           minLength:"1" doc:"Reading-order name." example:"Batman: Court of Owls"`
	Description    string `json:"description"    doc:"Reading-order description or notes."`
	Favorite       bool   `json:"favorite"       doc:"Whether this reading order is marked as a favorite." example:"true"`
	IsPublic       *bool  `json:"isPublic,omitempty" doc:"Whether users other than the creator may view this reading order. Defaults to true." example:"true"`
	CoverImageData string `json:"coverImageData,omitempty" doc:"Optional uploaded cover image as a data URL. The server resizes and stores it in the cover cache."`
}

type ReadingOrderRatingPayload struct {
	Rating float64 `json:"rating" minimum:"0" maximum:"5" doc:"Current user's rating from 1 to 5. Use 0 to clear the rating." example:"4"`
}

type ReadingOrderComicPayload struct {
	ComicID int    `json:"comicId" doc:"Local comic identifier to add to the reading order." example:"42"`
	Comment string `json:"comment" doc:"Optional note for this specific reading-order entry." example:"Read after issue 5"`
	Tags    string `json:"tags"    doc:"Comma-separated tags for this reading-order entry." example:"Main Story"`
}

type ReadingOrderEntryPayload struct {
	Type           string `json:"type"           enum:"comic,readingOrder,section" doc:"Entry type." example:"comic"`
	ComicID        int    `json:"comicId,omitempty"        doc:"Local comic identifier for comic entries." example:"42"`
	ReadingOrderID int    `json:"readingOrderId,omitempty" doc:"Child reading-order identifier for nested reading-order entries." example:"7"`
	Title          string `json:"title,omitempty"          doc:"Section title for section entries." example:"Main story"`
	Description    string `json:"description,omitempty"    doc:"Optional section description for section entries." example:"Read these issues before the tie-ins."`
	Comment        string `json:"comment,omitempty"        doc:"Optional note for comic or nested reading-order entries." example:"Read after issue 5"`
	Tags           string `json:"tags,omitempty"           doc:"Comma-separated tags for comic entries." example:"Main Story"`
}

type ReadingOrderComic struct {
	Comic
	Comment string `json:"comment" db:"comment" doc:"Per-entry reading-order note." example:"Tie-in"`
	Tags    string `json:"tags"    db:"tags"    doc:"Comma-separated entry tags synced from Metron or added locally." example:"Main Story"`
}

type ReadingOrderSection struct {
	Title       string `json:"title"       db:"title"       doc:"Section title." example:"Main story"`
	Description string `json:"description" db:"description" doc:"Optional context shown below the section title."`
}

type ReadingOrderEntry struct {
	Type         string               `json:"type" doc:"Entry type." enum:"comic,readingOrder,section" example:"comic"`
	Comic        *ReadingOrderComic   `json:"comic,omitempty" doc:"Comic entry payload when type is comic."`
	ReadingOrder *ReadingOrder        `json:"readingOrder,omitempty" doc:"Referenced reading order when type is readingOrder."`
	Section      *ReadingOrderSection `json:"section,omitempty" doc:"Section heading when type is section."`
	Comics       []ReadingOrderComic  `json:"comics,omitempty" doc:"Expanded comics when type is readingOrder."`
	Comment      string               `json:"comment,omitempty" doc:"Per-entry note for nested reading-order entries." example:"Read this crossover here"`
}

type ReadingOrderDetail struct {
	ReadingOrder
	Entries            []ReadingOrderEntry `json:"entries"            doc:"Direct reading-order entries in position order."`
	Comics             []ReadingOrderComic `json:"comics"             doc:"Comics in reading-order position order, including child reading-order comics."`
	ChildReadingOrders []ReadingOrder      `json:"childReadingOrders" doc:"Reading orders referenced by this reading order."`
}

type ReadingOrderListInput struct {
	Query     string `query:"q"        doc:"Case-insensitive text search across name and description." example:"batman"`
	Favorite  string `query:"favorite" doc:"Filter reading orders by favorite status. Use true or false." enum:"true,false" example:"true"`
	Started   string `query:"started" doc:"Filter reading orders by whether the current user formally started them. Use true or false." enum:"true,false" example:"true"`
	ComicID   int    `query:"comicId"  doc:"Filter reading orders to those containing a comic." example:"42"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,progress,rating,favoriteCount,startedCount" example:"name"`
	Direction string `query:"direction" doc:"Sort direction." enum:"asc,desc" example:"asc"`
	Limit     int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset    int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
}

type ReadingOrderInput struct {
	ID int `path:"id" doc:"Local reading-order identifier." example:"7"`
}

type UpdateReadingOrderRatingInput struct {
	ID   int `path:"id" doc:"Local reading-order identifier." example:"7"`
	Body ReadingOrderRatingPayload
}

type ReadingOrderCBLImportInput struct {
	Body struct {
		Filename string                      `json:"filename,omitempty" doc:"Original CBL filename, used as a fallback reading-order name for a single-file import." example:"Infinity Gauntlet.cbl"`
		Content  string                      `json:"content,omitempty"  doc:"CBL XML document content for a single-file import."`
		Parts    []ReadingOrderCBLImportPart `json:"parts,omitempty"    doc:"Two or more CBL part files that share a numbered Part or pt marker. All parts are combined into one reading order with a section per part."`
	}
}

type ReadingOrderCBLImportPart struct {
	Filename string `json:"filename,omitempty" doc:"Original CBL part filename." example:"[Marvel] CMRO Core Reading Order-Part 01.cbl"`
	Content  string `json:"content" minLength:"1" doc:"CBL XML document content for this part."`
}

type ReadingOrderCBLUnmatchedBook struct {
	Part     string `json:"part,omitempty" doc:"CBL part name when this book came from a multipart import." example:"[Marvel] CMRO Core Reading Order-Part 02"`
	Position int    `json:"position" doc:"One-based book position in the CBL file." example:"3"`
	Series   string `json:"series"   doc:"CBL book series attribute." example:"Batman"`
	Number   string `json:"number"   doc:"CBL book number attribute." example:"6"`
	Volume   string `json:"volume"   doc:"CBL book volume attribute." example:"2011"`
	Year     string `json:"year"     doc:"CBL book year attribute." example:"2012"`
	Reason   string `json:"reason"   doc:"Reason this malformed or ambiguous CBL book could not be matched or created." example:"multiple local comics matched"`
}

type ReadingOrderCBLImportResult struct {
	ReadingOrder   ReadingOrderDetail             `json:"readingOrder"   doc:"Created reading order with matched local comics."`
	MatchedCount   int                            `json:"matchedCount"   doc:"Number of CBL books matched to or created as local comics." example:"12"`
	UnmatchedCount int                            `json:"unmatchedCount" doc:"Number of malformed or ambiguous CBL books that could not be imported." example:"2"`
	Unmatched      []ReadingOrderCBLUnmatchedBook `json:"unmatched"      doc:"Malformed or ambiguous CBL books that could not be matched or created."`
}

type ReadingOrderCBLImportOutput struct {
	Body ReadingOrderCBLImportResult
}

type ReadingOrderCBLExport struct {
	Filename string `json:"filename" doc:"Suggested CBL download filename." example:"Batman Court of Owls.cbl"`
	Content  string `json:"content"  doc:"CBL XML document content."`
}

type ReadingOrderCBLExportOutput struct {
	Body ReadingOrderCBLExport
}

type ReadingOrderListOutput struct {
	PaginationHeaders
	Body []ReadingOrder
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

type CopyReadingOrderInput struct {
	ID int `path:"id" doc:"Local reading-order identifier to copy." example:"7"`
}

type SetReadingOrderComicsInput struct {
	ID   int `path:"id" doc:"Local reading-order identifier." example:"7"`
	Body struct {
		ComicIDs        []int                      `json:"comicIds,omitempty"        doc:"Comic IDs in reading order. Use comics to include comments." example:"[42,43]"`
		Comics          []ReadingOrderComicPayload `json:"comics,omitempty"          doc:"Comics in reading order with optional per-entry comments."`
		ReadingOrderIDs []int                      `json:"readingOrderIds,omitempty" doc:"Child reading-order IDs referenced by this reading order." example:"[7,8]"`
		Entries         []ReadingOrderEntryPayload `json:"entries,omitempty"         doc:"Ordered comic, child reading-order, and section entries."`
	}
}
