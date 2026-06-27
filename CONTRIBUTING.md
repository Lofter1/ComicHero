# Contributing

Thanks for considering a contribution to ComicHero.

## Development Setup

1. Install Go and Node/npm.
2. Install frontend dependencies:

   ```sh
   npm --prefix ui install
   ```

3. Copy `.env.example` to `.env` if you want local Metron imports.
4. Run checks before opening a pull request:

   ```sh
   make test
   ```

## Pull Request Guidelines

- Keep changes focused and describe the user-visible behavior.
- Add or update tests for backend behavior changes.
- Run `go test ./...` in `backend` for Go changes.
- Run `npm --prefix ui run build` for UI changes.
- Do not commit local databases, `.env` files, cached covers, build output, or temporary files.

## Code Style

- Follow existing Go and Vue patterns in the repository.
- Prefer small, direct UI improvements over broad redesigns.
- Keep self-hosted data private by default.

## Metron

Metron imports may require credentials and are rate-limited upstream. Avoid adding tests or features that depend on live Metron network calls.
