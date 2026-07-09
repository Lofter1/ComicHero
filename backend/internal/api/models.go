package api

import "net/http"

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

type User struct {
	ID            int    `json:"id"            db:"id"                doc:"Local user identifier." example:"1"`
	Name          string `json:"name"          db:"name"              doc:"Display name." example:"Justin"`
	Email         string `json:"email"         db:"email"             doc:"Email address used to log in." example:"reader@example.com"`
	EmailVerified bool   `json:"emailVerified" db:"email_verified"    doc:"Whether this user's email address has been verified." example:"true"`
	IsAdmin       bool   `json:"isAdmin"       db:"is_admin"          doc:"Whether the user can manage user permissions." example:"false"`
}

type UserMetronPermissions struct {
	Allowed     bool     `json:"allowed"     doc:"Whether this user can call ComicHero Metron endpoints." example:"true"`
	Scopes      []string `json:"scopes"      doc:"Allowed Metron scopes. Use * for all, or combine search, detail, import, and monitor." example:"search"`
	HourlyLimit int      `json:"hourlyLimit" minimum:"0" doc:"Maximum Metron endpoint calls per rolling hour. Use 0 for unlimited." example:"60"`
}

type UserAdminView struct {
	User              User                  `json:"user"              doc:"User account."`
	MetronPermissions UserMetronPermissions `json:"metronPermissions" doc:"Metron endpoint permissions for this user."`
}

type UserListOutput struct {
	Body []UserAdminView
}

type UserInvite struct {
	Token     string `json:"token"     doc:"Single-use invite token."`
	ExpiresAt string `json:"expiresAt" doc:"RFC3339 expiry time for this invite."`
}

type UserInviteOutput struct {
	Body UserInvite
}

type UpdateUserMetronPermissionsInput struct {
	ID   int `path:"id" doc:"Local user identifier." example:"2"`
	Body UserMetronPermissions
}

type UpdateUserAdminPayload struct {
	IsAdmin bool `json:"isAdmin" doc:"Whether the user should be an admin." example:"true"`
}

type UpdateUserAdminInput struct {
	ID   int `path:"id" doc:"Local user identifier." example:"2"`
	Body UpdateUserAdminPayload
}

type DeleteUserInput struct {
	ID int `path:"id" doc:"Local user identifier." example:"2"`
}

type UserAdminOutput struct {
	Body UserAdminView
}

type UpdateRegistrationModePayload struct {
	Mode string `json:"mode" doc:"Registration mode: invite_only requires invite tokens, open allows self-registration." enum:"invite_only,open" example:"invite_only"`
}

type UpdateRegistrationModeInput struct {
	Body UpdateRegistrationModePayload
}

type RegistrationModeOutput struct {
	Body UserStatus
}

type UpdatePublicAccessPayload struct {
	Enabled bool `json:"enabled" doc:"Whether anonymous read-only public access is enabled." example:"true"`
}

type UpdatePublicAccessInput struct {
	Body UpdatePublicAccessPayload
}

type PublicAccessOutput struct {
	Body UserStatus
}

type UserStatus struct {
	SetupRequired             bool                  `json:"setupRequired" doc:"Whether the app still needs single-user or multi-user setup." example:"false"`
	Mode                      string                `json:"mode,omitempty" doc:"Configured user mode: single or multi." enum:"single,multi" example:"single"`
	RegistrationMode          string                `json:"registrationMode" doc:"Configured registration mode: invite_only or open." enum:"invite_only,open" example:"invite_only"`
	PublicAccess              bool                  `json:"publicAccess" doc:"Whether anonymous read-only access is enabled." example:"false"`
	EmailVerificationRequired bool                  `json:"emailVerificationRequired" doc:"Whether login is blocked until email verification is completed." example:"false"`
	EmailVerificationEmail    string                `json:"emailVerificationEmail,omitempty" doc:"Email address waiting for verification." example:"reader@example.com"`
	User                      *User                 `json:"user,omitempty" doc:"Current user, when a session is active or single-user mode is enabled."`
	MetronPermissions         UserMetronPermissions `json:"metronPermissions" doc:"Current user's Metron endpoint permissions."`
}

type UserStatusOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      UserStatus
}

type UserStatistics struct {
	TotalComics            int     `json:"totalComics"          doc:"Total comics in the local library." example:"120"`
	ReadComics             int     `json:"readComics"           doc:"Comics marked read by the current user." example:"42"`
	UnreadComics           int     `json:"unreadComics"         doc:"Comics not yet marked read by the current user." example:"78"`
	ReadProgress           float64 `json:"readProgress"         doc:"Fraction of local comics marked read by the current user, from 0 to 1." minimum:"0" maximum:"1" example:"0.35"`
	FirstReadAt            string  `json:"firstReadAt,omitempty" doc:"Timestamp when the current user first marked a comic read." example:"2026-07-08T13:45:00Z"`
	LastReadAt             string  `json:"lastReadAt,omitempty"  doc:"Timestamp when the current user most recently marked a comic read." example:"2026-07-08T14:15:00Z"`
	DistinctReadSeries     int     `json:"distinctReadSeries"   doc:"Number of distinct series where the current user has read at least one comic." example:"8"`
	CompletedSeries        int     `json:"completedSeries"      doc:"Number of distinct series where every local comic is marked read by the current user." example:"2"`
	DistinctReadPublishers int     `json:"distinctReadPublishers" doc:"Number of distinct publishers where the current user has read at least one comic." example:"3"`
	AuthoredReadingOrders  int     `json:"authoredReadingOrders" doc:"Reading orders authored by the current user." example:"4"`
	StartedReadingOrders   int     `json:"startedReadingOrders" doc:"Reading orders where the current user has read at least one included comic." example:"5"`
	CompletedReadingOrders int     `json:"completedReadingOrders" doc:"Reading orders where every included comic is marked read by the current user." example:"2"`
	StartedArcs            int     `json:"startedArcs"          doc:"Story arcs where the current user has read at least one included comic." example:"3"`
	CompletedArcs          int     `json:"completedArcs"        doc:"Story arcs where every included comic is marked read by the current user." example:"1"`
	CharactersMet          int     `json:"charactersMet"        doc:"Distinct characters appearing in comics marked read by the current user." example:"17"`
}

type Achievement struct {
	ID          string  `json:"id"          doc:"Stable achievement identifier." example:"first-read"`
	Name        string  `json:"name"        doc:"Display name." example:"First Issue"`
	Description string  `json:"description" doc:"What earns this achievement." example:"Mark one comic as read."`
	Category    string  `json:"category"    doc:"Achievement category." example:"Reading"`
	Earned      bool    `json:"earned"      doc:"Whether the current user has earned this achievement." example:"true"`
	EarnedAt    string  `json:"earnedAt,omitempty" doc:"Timestamp when this achievement was earned, when derivable from read activity." example:"2026-07-08T14:15:00Z"`
	Progress    int     `json:"progress"    doc:"Current progress toward the achievement target." example:"42"`
	Target      int     `json:"target"      doc:"Progress needed to earn the achievement." example:"50"`
	Percent     float64 `json:"percent"     doc:"Progress fraction toward the target, from 0 to 1." minimum:"0" maximum:"1" example:"0.84"`
}

type UserStatisticsView struct {
	Statistics   UserStatistics `json:"statistics"   doc:"Current user's reading statistics."`
	Achievements []Achievement  `json:"achievements" doc:"Current user's earned and locked achievements."`
}

type UserStatisticsOutput struct {
	Body UserStatisticsView
}

type UserStatusInput struct {
	Session string `cookie:"comichero_session"`
}

type LogoutUserOutput struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

type LogoutUserInput struct {
	Session string `cookie:"comichero_session"`
}

type SetupUsersPayload struct {
	Mode     string `json:"mode" doc:"User mode to enable: single avoids login, multi enables registration and login." enum:"single,multi" example:"multi"`
	Name     string `json:"name,omitempty" doc:"Initial user name for multi-user mode. Existing read status is attached to this user." example:"Justin"`
	Email    string `json:"email,omitempty" doc:"Initial email address for multi-user login." format:"email" example:"reader@example.com"`
	Password string `json:"password,omitempty" doc:"Initial password for multi-user mode." example:"correct horse battery staple"`
}

type SetupUsersInput struct {
	Body SetupUsersPayload
}

type UserCredentialsPayload struct {
	Name                 string `json:"name,omitempty"                 doc:"Display name for registration." example:"Justin"`
	Email                string `json:"email"                          minLength:"1" format:"email" doc:"Email address used to log in." example:"reader@example.com"`
	EmailConfirmation    string `json:"emailConfirmation,omitempty"    format:"email" doc:"Repeated email address required for registration." example:"reader@example.com"`
	Password             string `json:"password"                       minLength:"6" doc:"Password." example:"correct horse battery staple"`
	PasswordConfirmation string `json:"passwordConfirmation,omitempty" minLength:"6" doc:"Repeated password required for registration." example:"correct horse battery staple"`
	InviteToken          string `json:"inviteToken,omitempty"          doc:"Invite token required for registration in multi-user mode."`
}

type RegisterUserInput struct {
	Body UserCredentialsPayload
}

type LoginUserInput struct {
	Body UserCredentialsPayload
}

type VerifyEmailPayload struct {
	Token string `json:"token" minLength:"1" doc:"Email verification token."`
}

type VerifyEmailInput struct {
	Body VerifyEmailPayload
}

type VerifyEmailLinkInput struct {
	Token string `query:"token" minLength:"1" doc:"Email verification token."`
}

type ResendEmailVerificationPayload struct {
	Email    string `json:"email"    minLength:"1" format:"email" doc:"Email address used to log in." example:"reader@example.com"`
	Password string `json:"password" minLength:"6" doc:"Password." example:"correct horse battery staple"`
}

type ResendEmailVerificationInput struct {
	Body ResendEmailVerificationPayload
}

type UpdateAccountPayload struct {
	Name            string `json:"name"                      minLength:"1" doc:"New display name." example:"Justin"`
	CurrentPassword string `json:"currentPassword,omitempty" doc:"Current password, required when changing password in multi-user mode."`
	NewPassword     string `json:"newPassword,omitempty"     doc:"New password. Leave empty to keep the current password."`
}

type UpdateAccountInput struct {
	Body UpdateAccountPayload
}

type DeleteAccountPayload struct {
	CurrentPassword string `json:"currentPassword,omitempty" doc:"Current password for multi-user account deletion."`
}

type DeleteAccountInput struct {
	Body DeleteAccountPayload
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
	Read           string `query:"read"           doc:"Filter comics by read status. Use true or false." enum:"true,false" example:"false"`
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
	Query     string `query:"q"        doc:"Case-insensitive text search across character names and aliases." example:"bat"`
	Favorite  string `query:"favorite" doc:"Filter characters by favorite status. Use true or false." enum:"true,false" example:"true"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,appearances,aliases,progress" example:"name"`
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
	Query     string `query:"q"        doc:"Case-insensitive text search across series names, publishers, years, and issue numbers." example:"batman"`
	Favorite  string `query:"favorite" doc:"Filter series by favorite status. Use true or false." enum:"true,false" example:"true"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,year,publisher,entries,progress" example:"name"`
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
	AuthorUserID        *int `json:"authorUserId,omitempty"              db:"author_user_id"         doc:"User who created or owns this reading order." example:"1"`

	Name        string  `json:"name"        db:"name"        doc:"Reading-order name." example:"Batman: Court of Owls"`
	Description string  `json:"description" db:"description" doc:"Reading-order description or notes."`
	Image       string  `json:"image"       db:"image"       doc:"Reading-list thumbnail image URL from Metron, when imported." format:"uri"`
	Favorite    bool    `json:"favorite"    db:"favorite"    doc:"Whether this reading order is marked as a favorite." example:"true"`
	Progress    float64 `json:"progress"    db:"progress"    doc:"Fraction of entries marked read, from 0 to 1." minimum:"0" maximum:"1" example:"0.5"`
	AuthorName  string  `json:"authorName"  db:"author_name" doc:"Display name of the reading-order author." example:"Default"`
	CanEdit     bool    `json:"canEdit"     db:"can_edit"    doc:"Whether the current user may edit this reading order." example:"true"`
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

type ReadingOrderEntryPayload struct {
	Type           string `json:"type"           enum:"comic,readingOrder" doc:"Entry type." example:"comic"`
	ComicID        int    `json:"comicId,omitempty"        doc:"Local comic identifier for comic entries." example:"42"`
	ReadingOrderID int    `json:"readingOrderId,omitempty" doc:"Child reading-order identifier for nested reading-order entries." example:"7"`
	Comment        string `json:"comment,omitempty"        doc:"Optional note for comic or nested reading-order entries." example:"Read after issue 5"`
	Tags           string `json:"tags,omitempty"           doc:"Comma-separated tags for comic entries." example:"Main Story"`
}

type ReadingOrderComic struct {
	Comic
	Comment string `json:"comment" db:"comment" doc:"Per-entry reading-order note." example:"Tie-in"`
	Tags    string `json:"tags"    db:"tags"    doc:"Comma-separated entry tags synced from Metron or added locally." example:"Main Story"`
}

type ReadingOrderEntry struct {
	Type         string             `json:"type" doc:"Entry type." example:"comic"`
	Comic        *ReadingOrderComic `json:"comic,omitempty" doc:"Comic entry payload when type is comic."`
	ReadingOrder *ReadingOrder      `json:"readingOrder,omitempty" doc:"Referenced reading order when type is readingOrder."`
	Comment      string             `json:"comment,omitempty" doc:"Per-entry note for nested reading-order entries." example:"Read this crossover here"`
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
	ComicID   int    `query:"comicId"  doc:"Filter reading orders to those containing a comic." example:"42"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,progress" example:"name"`
	Direction string `query:"direction" doc:"Sort direction." enum:"asc,desc" example:"asc"`
	Limit     int    `query:"limit"    doc:"Maximum rows to return, from 1 to 100." minimum:"1" maximum:"100" example:"50"`
	Offset    int    `query:"offset"   doc:"Zero-based row offset for pagination." minimum:"0" example:"0"`
}

type ReadingOrderInput struct {
	ID int `path:"id" doc:"Local reading-order identifier." example:"7"`
}

type ReadingOrderCBLImportInput struct {
	Body struct {
		Filename string `json:"filename,omitempty" doc:"Original CBL filename, used as a fallback reading-order name." example:"Infinity Gauntlet.cbl"`
		Content  string `json:"content"            minLength:"1" doc:"CBL XML document content."`
	}
}

type ReadingOrderCBLUnmatchedBook struct {
	Position int    `json:"position" doc:"One-based book position in the CBL file." example:"3"`
	Series   string `json:"series"   doc:"CBL book series attribute." example:"Batman"`
	Number   string `json:"number"   doc:"CBL book number attribute." example:"6"`
	Volume   string `json:"volume"   doc:"CBL book volume attribute." example:"2011"`
	Year     string `json:"year"     doc:"CBL book year attribute." example:"2012"`
	Reason   string `json:"reason"   doc:"Reason this CBL book could not be matched to a local comic." example:"no local comic matched"`
}

type ReadingOrderCBLImportResult struct {
	ReadingOrder   ReadingOrderDetail             `json:"readingOrder"   doc:"Created reading order with matched local comics."`
	MatchedCount   int                            `json:"matchedCount"   doc:"Number of CBL books matched to local comics." example:"12"`
	UnmatchedCount int                            `json:"unmatchedCount" doc:"Number of CBL books that could not be matched." example:"2"`
	Unmatched      []ReadingOrderCBLUnmatchedBook `json:"unmatched"      doc:"CBL books that were left out because no local comic matched."`
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
	Query     string `query:"q"        doc:"Case-insensitive text search across name and description." example:"batman"`
	Favorite  string `query:"favorite" doc:"Filter arcs by favorite status. Use true or false." enum:"true,false" example:"true"`
	ComicID   int    `query:"comicId"  doc:"Filter arcs to those containing a comic." example:"42"`
	Sort      string `query:"sort"     doc:"Sort field." enum:"name,progress" example:"name"`
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
		Entries         []ReadingOrderEntryPayload `json:"entries,omitempty"         doc:"Ordered comic and child reading-order entries."`
	}
}
