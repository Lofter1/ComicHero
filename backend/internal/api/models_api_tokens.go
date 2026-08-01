package api

// Scopes that can be granted to an API token. These map one-to-one onto the
// individual capabilities an external service (a reading server, collection
// manager, etc.) might need - not a blanket read/write split - so a token
// can be limited to, say, only searching and fetching the next comic,
// without also being able to mark things read.
const (
	apiTokenScopeReadingOrdersSearch = "readingOrders:search"
	apiTokenScopeReadingOrdersRead   = "readingOrders:read"
	apiTokenScopeReadingOrdersNext   = "readingOrders:next"
	apiTokenScopeReadingOrdersStart  = "readingOrders:start"
	apiTokenScopeComicsMarkRead      = "comics:markRead"
)

var validAPITokenScopes = map[string]bool{
	apiTokenScopeReadingOrdersSearch: true,
	apiTokenScopeReadingOrdersRead:   true,
	apiTokenScopeReadingOrdersNext:   true,
	apiTokenScopeReadingOrdersStart:  true,
	apiTokenScopeComicsMarkRead:      true,
}

// APIToken is the public (non-secret) representation of a token, returned
// by list/create responses. The raw token value is only ever returned once,
// at creation time, in APITokenCreated.
type APIToken struct {
	ID          int      `json:"id"          db:"id"           doc:"Local API token identifier." example:"1"`
	Name        string   `json:"name"        db:"name"         doc:"User-chosen label for this token." example:"Tachiyomi bridge"`
	TokenPrefix string   `json:"tokenPrefix" db:"token_prefix" doc:"First few characters of the token, shown so the token can be recognized without revealing it." example:"ch_pat_3f2a"`
	Scopes      []string `json:"scopes"      doc:"Granted permission scopes." example:"readingOrders:search"`
	CreatedAt   string   `json:"createdAt"   db:"created_at"   doc:"When the token was created." example:"2026-08-01T10:00:00Z"`
	LastUsedAt  string   `json:"lastUsedAt"  db:"last_used_at" doc:"When the token was last used to authenticate a request. Empty if never used." example:"2026-08-01T12:00:00Z"`
	ExpiresAt   string   `json:"expiresAt"   db:"expires_at"   doc:"When the token expires. Empty if it does not expire." example:""`
	RevokedAt   string   `json:"revokedAt"   db:"revoked_at"   doc:"When the token was revoked. Empty if still active." example:""`
}

type apiTokenRow struct {
	APIToken
	ScopesRaw string `db:"scopes"`
}

type APITokenListOutput struct {
	Body []APIToken
}

type CreateAPITokenPayload struct {
	Name      string   `json:"name"      minLength:"1" doc:"User-chosen label for this token." example:"Tachiyomi bridge"`
	Scopes    []string `json:"scopes"    doc:"Permission scopes to grant. One or more of readingOrders:search, readingOrders:read, readingOrders:next, readingOrders:start, comics:markRead." example:"readingOrders:search"`
	ExpiresAt string   `json:"expiresAt,omitempty" doc:"Optional RFC3339 expiry. Omit for a token that never expires." example:"2027-08-01T00:00:00Z"`
}

type CreateAPITokenInput struct {
	Body CreateAPITokenPayload
}

// APITokenCreated is returned exactly once, at creation time. Token holds
// the raw secret value - it is never stored and can never be retrieved
// again, so the caller must save it immediately.
type APITokenCreated struct {
	APIToken
	Token string `json:"token" doc:"The raw API token. Shown only once - store it now, it cannot be retrieved again." example:"ch_pat_3f2a9c8b1d2e4f5061728394a5b6c7d8"`
}

type CreateAPITokenOutput struct {
	Body APITokenCreated
}

type RevokeAPITokenInput struct {
	ID int `path:"id" doc:"Local API token identifier." example:"1"`
}

// NextComicOutput reports the next unread comic in a reading order, in
// reading-order position order (flattening nested reading orders the same
// way the reading order detail endpoint does).
type NextComicOutput struct {
	Body struct {
		ReadingOrderID int                `json:"readingOrderId" doc:"Reading order this result is for." example:"7"`
		Done           bool               `json:"done"            doc:"True when every comic in the reading order has already been marked read." example:"false"`
		Comic          *ReadingOrderComic `json:"comic,omitempty" doc:"The next unread comic, in reading order. Omitted when done is true."`
	}
}
