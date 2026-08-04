// Package metron provides ComicHero's own domain-shaped view of the
// upstream Metron API: it maps Metron's JSON responses onto ComicHero's
// Issue/MetronArc/Series/ReadingList types (see mapping.go) and adds
// import/sync-oriented behavior such as proactive rate-limit backoff and a
// diagnostics request log.
//
// The actual HTTP transport, authentication, and base-URL handling are
// delegated to backend/metron, a general-purpose Metron API client that
// mirrors Metron's schema directly. Reach for backend/metron instead of
// this package for anything that isn't specific to ComicHero's own import
// and persistence workflows, which belong in package api.
package metron
