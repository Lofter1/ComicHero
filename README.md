# ComicHero

ComicHero is a self-hosted reading-order tracker for comics. It helps you build reading orders, track read progress, and optionally import metadata from [Metron](https://metron.cloud/).

> [!IMPORTANT]
> Update to ComicHero 1.5.1 or later. Earlier versions can send broken conditional requests to Metron that may cause accounts to be falsely flagged as duplicates and blocked.

[![Join the ComicHero community on Discord](https://img.shields.io/badge/Join_the_community-Discord-5865F2?logo=discord&logoColor=white)](https://discord.gg/GebUwAVP)

Chat with other ComicHero users, ask questions, share feedback, and follow development in the community Discord.

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

Metron credentials are optional and only needed if you want to import comics, reading lists, and series from [Metron](https://metron.cloud/). Every install option below supports setting them either as environment variables (`METRON_USERNAME` / `METRON_PASSWORD`) or via a `.env` file — pick whichever fits the option you're using.

### Option 1: Docker (recommended)

Pull the published image and run it, passing Metron credentials as environment variables if you want them:

```sh
docker run -d \
  --name comichero \
  -p 8080:8080 \
  -v comichero-data:/data \
  -e METRON_USERNAME=your-metron-username \
  -e METRON_PASSWORD=your-metron-password \
  ghcr.io/lofter1/comichero:latest
```

Omit the two `-e` lines if you don't need Metron import. Open `http://localhost:8080`. SQLite data and cached covers live in the `comichero-data` volume.

Images are published on every tagged release (`ghcr.io/lofter1/comichero:v1.2.3`) and `latest` always points at the most recent release. See the [Releases page](https://github.com/Lofter1/ComicHero/releases) for available tags.

### Option 2: Docker Compose

1. Copy the example environment file:

   ```sh
   cp .env.example .env
   ```

2. Open `.env` and, if you want Metron import, fill in:

   ```
   METRON_USERNAME=your-metron-username
   METRON_PASSWORD=your-metron-password
   ```

3. Start the app:

   ```sh
   docker compose up
   ```

   This pulls `ghcr.io/lofter1/comichero:latest` as defined in `compose.yaml`. To build from a local checkout instead, run `docker compose up --build` (see the commented-out `build: .` line in `compose.yaml`).

4. Open `http://localhost:8080`.

### Option 3: Prebuilt binary

Each [release](https://github.com/Lofter1/ComicHero/releases) includes standalone binaries for Linux and macOS (amd64 and arm64), with the frontend already embedded, plus a `.env.example` template. No Docker, Node, or Go required.

```sh
curl -LO https://github.com/Lofter1/ComicHero/releases/latest/download/comichero_<version>_linux_amd64.tar.gz
tar -xzf comichero_<version>_linux_amd64.tar.gz
cd comichero_<version>_linux_amd64
cp .env.example .env   # optional: fill in METRON_USERNAME / METRON_PASSWORD
./comichero
```

Replace `linux_amd64` with `linux_arm64`, `darwin_amd64`, or `darwin_arm64` as needed. The binary reads `.env` from its own directory automatically; see [Configuration](#configuration) for all available variables.

### Option 4: Build from source

Requirements: Go matching `backend/go.mod`, Node.js 24 LTS, npm.

```sh
cp .env.example .env   # optional: fill in METRON_USERNAME / METRON_PASSWORD
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
| `COOKIE_SECURE`   | auto-detected              | Sets the `Secure` flag on session cookies. Auto-detected from the request (direct TLS, or `X-Forwarded-Proto: https` from a reverse proxy) unless explicitly set to `true` or `false`. |
| `APP_BASE_URL`    | `http://localhost:8080`    | Public base URL used in account email links. Set this to your HTTPS origin for public instances. |
| `SMTP_HOST`       | empty                      | SMTP host for sending account emails such as verification and password reset. If unset, links are logged instead of sent. |
| `SMTP_PORT`       | `587`                      | SMTP port. |
| `SMTP_USERNAME`   | empty                      | Optional SMTP username. |
| `SMTP_PASSWORD`   | empty                      | Optional SMTP password. |
| `SMTP_FROM`       | `SMTP_USERNAME` or `noreply@localhost` | From address for email verification messages. Many providers require this to match the authenticated SMTP account. |
| `METRON_BASE_URL` | `https://metron.cloud/api` | Metron API base URL.                                |
| `METRON_USERNAME` | empty                      | Optional Metron username.                           |
| `METRON_PASSWORD` | empty                      | Optional Metron password.                           |

### User registration modes

ComicHero asks you to choose single-user or multi-user mode on first run. In multi-user mode, admins can choose how new accounts are created:

- `invite_only` is the default. New registrations require a valid single-use invite token generated by an admin.
- `open` allows anyone who can reach the server to register with a name, email, and password, without an invite token. These users must verify their email address before they receive a session.

Open registration is intended only for instances you deliberately expose for self-service signup. ComicHero uses a shared-library model: comics, arcs, series, characters, and reading orders are not isolated per account. A verified new account can read and write the shared library data according to the app's normal user capabilities, so leave registration on `invite_only` unless you understand and accept that exposure. For public instances, serve ComicHero over HTTPS and set `APP_BASE_URL` to the HTTPS origin and configure SMTP; `COOKIE_SECURE` is detected automatically from the request but can be forced with an explicit `true`/`false` if you're behind a proxy that doesn't set `X-Forwarded-Proto`. Admins can change the mode and remove unwanted accounts from the user management screen.

Users can request a password reset from the login screen. Reset links use the same SMTP settings and expire after 30 minutes.

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
