# Contributing

Thanks for considering a contribution to ComicHero.

## Development Setup

1. Install Go and Node/npm.
2. Install frontend dependencies:

   ```sh
   npm --prefix ui install
   ```

3. Copy `.env.example` to `.env` if you want local Metron imports.
4. If you use VS Code, install the workspace recommended extensions when prompted. They enable Vue, ESLint, Prettier, and Go integration from the checked-in `.vscode` settings.
5. Run checks before opening a pull request:

   ```sh
   make test
   ```

## Pull Request Guidelines

- Keep changes focused and describe the user-visible behavior.
- Add or update tests for backend behavior changes.
- Run `go test ./...` in `backend` for Go changes.
- Run `npm --prefix ui run build` for UI changes.
- Run `make lint`
- Do not commit local databases, `.env` files, cached covers, build output, or temporary files.

## Code Style

- Follow existing Go and Vue patterns in the repository.
- Read [the backend architecture guide](backend/ARCHITECTURE.md) before adding Go packages, routes, or database migrations.
- VS Code users get frontend ESLint validation, Prettier formatting, and ESLint fixes on save from the shared workspace settings.
- Prefer small, direct UI improvements over broad redesigns.
- A component owns its appearance: keep semantic classes and scoped `@apply` styles in that component. Parents may control placement and layout, but must not reach into a child to restyle its internals. Add a documented prop or variant when callers need a supported visual difference.
- Keep one-caller UI in its feature directory. Move it to `ui/src/shared` only after at least two features share the same clear contract.
- Keep self-hosted data private by default.

## Metron

Metron imports may require credentials and are rate-limited upstream. Avoid adding tests or features that depend on live Metron network calls.
