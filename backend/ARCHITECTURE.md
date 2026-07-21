# Backend architecture

This guide explains where backend code belongs and how to extend it without
having to rediscover the package structure first.

## Package map

```text
backend/
├── main.go                 process entry point only
└── internal/
    ├── app/                application assembly, HTTP routing, static files, logging
    ├── config/             environment and .env parsing
    ├── api/                ComicHero HTTP API and application use cases
    ├── db/                 SQLite setup, migrations, and legacy schema compatibility
    ├── metron/             typed client for the upstream Metron API
    └── static/             embedded frontend filesystem
```

The dependency direction is intentionally one-way:

```text
main -> config + app -> api + db + metron + static
                         api -> metron
```

`main.go` should remain small. Concrete dependencies are created in
`internal/app`; feature code should never import `main` or assemble the whole
application itself.

## Application startup

`config.FromEnv` creates one process-level configuration value. `app.Run` opens
the database and cover cache, while `app.buildHandler` assembles middleware,
API routes, background workers, cover delivery, and the frontend fallback.
Adapter-owned defaults, such as Metron's base URL, remain in the adapter rather
than being duplicated in process configuration.

When adding process infrastructure:

1. Add its setting to `config.Config` and `config.FromEnv`.
2. Construct it in `internal/app`.
3. Pass the narrow dependency into the API registration function that needs it.
4. Return cleanup work from the assembly layer when the dependency owns a
   goroutine, file, or connection.

Avoid reading new environment variables from feature handlers. The existing
email and cookie settings are request-time exceptions and should stay localized
to the user/email modules until they are moved behind an injected service.

## API organization

`internal/api` is organized by product feature. A feature's public Huma route
registration stays close to its handlers, but large features use responsibility
suffixes:

- `<feature>_routes.go`: route metadata and dependency binding only.
- `<feature>_list.go`: list queries, filters, sorting, and pagination.
- `<feature>_detail.go`: detail hydration and read operations.
- `<feature>_mutations.go`: create, update, delete, and transaction workflows.
- `<feature>_entries.go`: ordered child-entry validation and persistence.
- `models_<feature>.go`: request, response, and persisted view types.
- `<feature>_test.go`: tests for that responsibility rather than one global test file.

Not every feature needs every suffix. Keep a feature in one file while it is
small; split it when distinct responsibilities become hard to scan. Do not add
generic `helpers.go` or `utils.go` files. Give shared code a precise name such
as `sql_helpers.go`, `content_starts.go`, or `metron_import_options.go`.

The API currently remains one Go package because transactions and hydrated
views intentionally cross comics, series, arcs, characters, and reading orders.
Creating subpackages solely to reduce file count would force those internals to
be exported without producing a useful domain boundary. Introduce a new package
only when it has a small public contract and can be tested independently, as
with `config`, `db`, and `metron`.

The CBL repository importer is the reference split for a stateful background
workflow. `readingorders_repository_sync.go` owns orchestration,
`readingorders_repository_settings.go` owns persisted configuration and
validation, `readingorders_repository_github.go` owns remote repository access,
`readingorders_repository_metron.go` owns ambiguous issue resolution,
`readingorders_repository_status.go` owns snapshots/subscriptions, and
`readingorders_repository_routes.go` owns the HTTP/SSE contract. Keep new sync
behavior with the responsibility it changes rather than growing the
orchestrator again.

### Adding an endpoint

1. Put request and response types in `models_<feature>.go`.
2. Register the Huma operation in the feature's route function.
3. Put query or mutation logic in the matching responsibility file.
4. Reuse `currentUserID`, pagination, content-preference, and SQL query helpers
   rather than duplicating authentication or filter behavior.
5. Add a focused test beside that feature. Route metadata belongs in
   `docs_test.go`; behavior belongs in the feature test.

Route functions should describe HTTP concerns and bind dependencies. They
should delegate the actual operation to a named function so the behavior can be
tested without constructing an HTTP server.

## Metron client

`internal/metron` is an upstream adapter, not ComicHero business logic:

- `client.go` owns client construction and observable client state.
- `transport.go` owns HTTP, authentication, pagination, request logging, and
  upstream rate-limit handling.
- `mapping.go` converts Metron's flexible JSON payloads to typed values.
- resource files (`issues.go`, `series.go`, `reading_lists.go`, and so on) expose
  the operations available for each upstream resource.
- `types.go` contains the adapter's public data contract.

ComicHero persistence and import decisions belong in `internal/api`, not in the
Metron transport package. Tests use local HTTP servers; they must never require
live Metron credentials.

## Database changes

Goose migrations in `internal/db/migrations` are the history and source of truth
for schema changes. Always add a new numbered migration; never edit an applied
migration.

The `schema_*.go` files are compatibility checks for databases created by older
ComicHero versions. Keep additions feature-scoped:

- `schema_users.go` for accounts, sessions, permissions, and user state.
- `schema_reading_orders.go` for reading-order ownership and ratings.
- `schema_content.go` for comics and series relationships.
- `schema_helpers.go` for schema introspection only.

If an existing installation could be missing a column or index, update both the
new migration and the matching compatibility function. Add a migration test in
`internal/db` for upgrade-sensitive changes.

## Verification

From `backend/`:

```sh
gofmt -w .
golangci-lint run ./...
go test ./...
go build ./...
```

Some API and Metron tests use `httptest.NewServer`, which needs permission to
bind a local loopback port. A restricted sandbox can reject that bind even when
the code compiles; `go test -run '^$' ./...` remains a useful compile-only check,
but the full suite still needs to pass in a normal development environment.
