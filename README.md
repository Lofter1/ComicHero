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

## Installation

### Option 1: Docker (recommended)

Pull the published image and run it directly:

```sh
docker run -d \
  --name comichero \
  -p 8080:8080 \
  -v comichero-data:/data \
  ghcr.io/lofter1/comichero:latest
```

Open `http://localhost:8080`. SQLite data and cached covers live in the `comichero-data` volume.

Images are published on every tagged release (`ghcr.io/lofter1/comichero:v1.2.3`) and `latest` always points at the most recent release. See the [Releases page](https://github.com/Lofter1/ComicHero/releases) for available tags.

### Option 2: Docker Compose

1. Copy the example environment file and optionally add Metron credentials:

   ```sh
   cp .env.example .env
   ```

2. Start the app:

   ```sh
   docker compose up
   ```

   This pulls `ghcr.io/lofter1/comichero:latest` as defined in `compose.yaml`. To build from a local checkout instead, run `docker compose up --build` (see the commented-out `build: .` line in `compose.yaml`).

3. Open `http://localhost:8080`.

### Option 3: Prebuilt binary

Each [release](https://github.com/Lofter1/ComicHero/releases) includes standalone binaries for Linux and macOS (amd64 and arm64), with the frontend already embedded. No Docker, Node, or Go required.

```sh
curl -LO https://github.com/Lofter1/ComicHero/releases/latest/download/comichero_<version>_linux_amd64.tar.gz
tar -xzf comichero_<version>_linux_amd64.tar.gz
cd comichero_<version>_linux_amd64
./comichero
```

Replace `linux_amd64` with `linux_arm64`, `darwin_amd64`, or `darwin_arm64` as needed. Configure via environment variables (see [Configuration](#configuration)) or a `.env` file next to the binary.

### Option 4: Build from source

Requirements: Go matching `backend/go.mod`, Node.js 24 LTS, npm.

```sh
make install-ui
make build-standalone
./dist/comichero
```

This builds the frontend, embeds it into the Go binary, and produces a single standalone executable at `dist/comichero` — the same artifact published in releases.

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
| `STATIC_DIR`      | _(embedded)_                | Optional: serve the frontend from this directory instead of the copy embedded in the binary. Useful for local frontend development. |
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
