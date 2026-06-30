# ComicHero

ComicHero is a self-hosted reading-order tracker for comics. It helps you build reading orders, track read progress, and optionally import metadata from [Metron](https://metron.cloud/).

## Features

- Build ordered reading lists with per-entry notes.
- Track read/unread progress across comics and reading orders.
- Search Metron for comics, series, and reading lists.
- Import Metron comics, reading lists, and series in background jobs.
- Run as a single Docker container with SQLite storage.

## Stack

- Backend: Go, chi, Huma, sqlx, SQLite, goose migrations.
- Frontend: Vue 3 and Vite.
- Packaging: Docker and Docker Compose.

## Quick Start With Docker Compose

1. Copy the example environment file:

   ```sh
   cp .env.example .env
   ```

2. Optionally add Metron credentials to `.env`.

3. Start the app:

   ```sh
   docker compose up --build
   ```

4. Open `http://localhost:8080`.

SQLite data is stored in the `comichero-data` Docker volume.

## Local Development

Requirements:

- Go matching `backend/go.mod`
- Node.js 24 LTS
- npm

Install frontend dependencies:

```sh
npm --prefix ui install
```

Run backend tests:

```sh
make test-backend
```

Run frontend build verification:

```sh
make test-ui
```

Run both:

```sh
make test
```

Start the backend and frontend separately:

```sh
make dev-backend
make dev-ui
```

The Vite dev server proxies `/api` and `/covers` to the Go backend.

## Configuration

The backend reads environment variables from the process environment and, when present, `.env` files.

| Variable          | Default                    | Description                                         |
| ----------------- | -------------------------- | --------------------------------------------------- |
| `PORT`            | `8080`                     | HTTP port for the Go server.                        |
| `DB_PATH`         | `./data/comicorder.db`     | SQLite database path.                               |
| `STATIC_DIR`      | `./public`                 | Directory for built frontend assets.                |
| `COVER_CACHE_DIR` | `./public/covers`          | Directory where downloaded cover images are cached. |
| `METRON_BASE_URL` | `https://metron.cloud/api` | Metron API base URL.                                |
| `METRON_USERNAME` | empty                      | Optional Metron username.                           |
| `METRON_PASSWORD` | empty                      | Optional Metron password.                           |

## API Documentation

When the backend is running, Huma exposes interactive API documentation at:

```text
http://localhost:8080/api/docs
```

## Roadmap

- Import and export library data for backups, migrations, and sharing.
- Optional login and user management for deployments that need multiple users.
- Edit views for all stored data.
- Sync read status with Metron when running in single-user mode.
- Open Source like approach on Reading Orders

## Data and Privacy

ComicHero is intended for self-hosted personal reading-order data. The repository does not include comic metadata, cover images, credentials, or a database. Local runtime data paths such as `backend/data`, `tmp`, and cached covers are ignored by Git.

Metron data and cover images may be subject to Metron's terms. Use your own API credentials and respect upstream rate limits.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

See [SECURITY.md](SECURITY.md).

## License

ComicHero is licensed under the [MIT License](LICENSE).
