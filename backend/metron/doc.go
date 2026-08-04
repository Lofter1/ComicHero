// Package metron is a general-purpose Go client for the Metron comic-book
// database API (https://metron.cloud). It mirrors the Metron OpenAPI schema
// resource-for-resource and endpoint-for-endpoint: every path documented at
// /api/schema/ has a corresponding typed method on Client.
//
// The package has no knowledge of ComicHero's database or domain model - it
// only translates Go calls into Metron HTTP requests and Metron JSON
// responses into Go structs, the same role backend/comicvine plays for the
// Comic Vine API. Application-specific mapping from Metron's schema onto
// ComicHero's own types lives in backend/internal/metron, which uses this
// package as its HTTP/transport layer.
//
// # Authentication
//
// Metron supports three schemes: HTTP Basic Auth, session cookies, and
// bearer API tokens (Knox). Configure whichever is available via
// [WithBasicAuth], [WithCookie], or [WithToken]; a token takes priority when
// several are configured, matching Metron's own precedence.
//
// # Pagination
//
// List endpoints return a page at a time via [PagedResponse]. Use a
// service's List method for a single page, or its All method to
// transparently walk every page.
package metron
