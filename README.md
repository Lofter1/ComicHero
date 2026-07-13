# ComicHero

ComicHero is a self-hosted reading-order tracker for comics. Build curated reading orders, follow your progress across series and story arcs, and enrich tracked comics with metadata from [Metron](https://metron.cloud/).

> [!IMPORTANT]
> Use ComicHero 1.5.1 or later. Earlier releases can send malformed conditional requests to Metron, which may cause accounts to be incorrectly flagged as duplicates and blocked.

[![Latest release](https://img.shields.io/github/v/release/Lofter1/ComicHero?label=release)](https://github.com/Lofter1/ComicHero/releases/latest)
[![Container image](https://img.shields.io/badge/container-ghcr.io-2496ED?logo=docker&logoColor=white)](https://github.com/Lofter1/ComicHero/pkgs/container/comichero)
[![Join the ComicHero community on Discord](https://img.shields.io/badge/Join_the_community-Discord-5865F2?logo=discord&logoColor=white)](https://discord.gg/GebUwAVP)

Join the Discord to ask questions, share feedback and reading orders, and follow development.

## What ComicHero can do

- Create, edit, reorder, rate, and favorite reading orders with per-entry notes and tags.
- Track comics as unread, read, or skipped with progress calculated per user.
- Browse comics by reading order, series, story arc, and character.
- Mark reading orders, series, arcs, and characters as started or favorite.
- Continue active reading from the dashboard and review statistics and achievements.
- Search and import comics, reading lists, series, arcs, and characters from Metron.
- Run scheduled Metron discovery jobs for new comics and reading lists.
- Fill incomplete comic metadata automatically while respecting configurable call limits and retry cooldowns.
- Choose single-user or multi-user setup, invite users, or enable open registration.
- Optionally give visitors read-only access to shared ComicHero content.
- Run as a single container or standalone binary backed by SQLite.

ComicHero shares comics, reading orders, arcs, series, and characters across the instance, while reading state, favorites, ratings, and progress are associated with individual users.

## Quick start with Docker

Metron credentials are optional. Omit the two `METRON_*` variables if you only want to manage data manually.

```sh
docker run -d \
  --name comichero \
  --restart unless-stopped \
  -p 8080:8080 \
  -v comichero-data:/data \
  -e METRON_USERNAME=your-metron-username \
  -e METRON_PASSWORD=your-metron-password \
  ghcr.io/lofter1/comichero:latest
```

Open `http://localhost:8080` and complete the first-run setup. ComicHero asks whether the instance should use single-user or multi-user mode and creates its initial account.

The named volume stores the SQLite database and cached cover images. Images are published for tagged releases, and `latest` points to the newest release. Pin a version such as `ghcr.io/lofter1/comichero:1.5.1` when reproducible deployments matter.

## Installation options

### Docker Compose

```sh
cp .env.example .env
docker compose up -d
```

Add your Metron credentials to `.env` before starting if you want imports. The included [compose.yaml](compose.yaml) publishes ComicHero on port `8080` and persists `/data` in the `comichero-data` volume.

To build the container from your checkout instead of pulling the published image, uncomment `build: .` in `compose.yaml` and run:

```sh
docker compose up -d --build
```

### Prebuilt binary

Each [release](https://github.com/Lofter1/ComicHero/releases) provides standalone Linux and macOS binaries for amd64 and arm64. The web interface is embedded, so Node.js is not required at runtime.

```sh
curl -LO https://github.com/Lofter1/ComicHero/releases/latest/download/comichero_<version>_linux_amd64.tar.gz
tar -xzf comichero_<version>_linux_amd64.tar.gz
cd comichero_<version>_linux_amd64
cp .env.example .env
./comichero
```

Replace `linux_amd64` with `linux_arm64`, `darwin_amd64`, or `darwin_arm64` as appropriate. The binary reads `.env` from its working directory.

### Build from source

Requirements are Go matching `backend/go.mod`, Node.js 24 LTS, and npm.

```sh
npm --prefix ui install
make build-standalone
./dist/comichero
```

This builds the Vue frontend, embeds it in the Go application, and writes a standalone executable to `dist/comichero`.

## Configuration

ComicHero reads the process environment and `.env` files in the current or parent directory. Process environment variables take precedence.

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port used by the server. |
| `DB_PATH` | `./data/comicorder.db` | SQLite database file. Parent directories are created automatically. |
| `COVER_CACHE_DIR` | `./public/covers` | Storage directory for downloaded and optimized cover images. |
| `ACCESS_LOG_PATH` | `./data/access.log` | Append-only JSON Lines HTTP access log. Set it explicitly to an empty value to disable file logging. |
| `STATIC_DIR` | embedded frontend | Optional directory from which to serve frontend files instead of the embedded build. |
| `SHOW_VERSION` | `true` | Show the running ComicHero version below the sidebar branding. Set to `false` to hide it. |
| `CHECK_FOR_UPDATES` | `true` | Check GitHub for a newer stable release and show an in-app notice when one is available. |
| `METRON_BASE_URL` | `https://metron.cloud/api` | Metron API base URL. |
| `METRON_USERNAME` | empty | Metron username used for search, import, and maintenance jobs. |
| `METRON_PASSWORD` | empty | Metron password. |
| `APP_BASE_URL` | `http://localhost:<PORT>` | Public origin used in verification and password-reset links. |
| `COOKIE_SECURE` | auto-detected | Force session cookies to use or omit `Secure` with `true` or `false`. Otherwise TLS and `X-Forwarded-Proto` are detected. |
| `SMTP_HOST` | empty | SMTP server for verification and password-reset emails. Links are logged when SMTP is unset. |
| `SMTP_PORT` | `587` | SMTP server port. |
| `SMTP_USERNAME` | empty | Optional SMTP username. |
| `SMTP_PASSWORD` | empty | Optional SMTP password. |
| `SMTP_FROM` | SMTP username or `noreply@localhost` | Sender address for account email. |

For a public deployment, put ComicHero behind HTTPS, set `APP_BASE_URL` to its public HTTPS origin, and configure SMTP. Ensure the reverse proxy sends `X-Forwarded-Proto: https`, or set `COOKIE_SECURE=true` explicitly.

## Accounts and access

The first-run wizard offers two modes:

- **Single-user** creates a personal instance with one account.
- **Multi-user** enables account administration and per-user reading progress.

Multi-user registration defaults to `invite_only`, where an administrator generates single-use invitation links. Administrators can instead enable `open` registration; new users then need to verify their email address before receiving a session. Password-reset links expire after 30 minutes.

Public read-only access can be enabled separately. Because comics, reading orders, and related content are shared across the instance, only enable open registration or public access when that exposure is intentional.

## Metron integration

[Metron](https://metron.cloud/) supplies optional comic metadata and cover images. With credentials configured, ComicHero can:

- search for and import individual records;
- import complete Metron series and reading lists in background jobs;
- discover newly modified comics and reading lists on a daily, weekly, or monthly schedule;
- repair missing publisher, cover, cover-date, and description fields on a schedule;
- apply call limits, minimum request intervals, and cooldowns for incomplete records.

Imports and scans are rate-limited upstream. Use your own Metron account, choose conservative schedules, and review Metron’s terms before enabling automation.

## Data, backups, and upgrades

The database contains comic metadata, reading orders, accounts, reading progress, settings, and job state. Cover images are cached separately. In the standard container deployment, both live under `/data` in the `comichero-data` volume.

Back up both the SQLite database and cover directory. For a simple consistent backup, stop ComicHero before copying its data volume. Never commit the database, `.env`, or cached covers to Git.

Database migrations run automatically when ComicHero starts. Before upgrading, back up `/data`, then pull the desired image and recreate the container:

```sh
docker compose pull
docker compose up -d
```

Use the [release notes](https://github.com/Lofter1/ComicHero/releases) to check for version-specific instructions.

## Local development

Install dependencies and run all checks:

```sh
npm --prefix ui install
make test
make lint
```

Run the backend and Vite development server together with `make dev`, or separately:

```sh
make dev-backend
make dev-ui
```

The Vite server runs at `http://localhost:5173` and proxies `/api` and `/covers` to the Go backend. Additional development guidance is in [CONTRIBUTING.md](CONTRIBUTING.md).

## API and health checks

With ComicHero running:

- Interactive API documentation: `http://localhost:8080/api/docs`
- Health endpoint: `http://localhost:8080/healthz`

## Access logs

ComicHero writes every HTTP request to `ACCESS_LOG_PATH` as one JSON object per line. Entries include the timestamp, method, path without its query string, response status, duration, response size, remote address, forwarded address, and user agent. This format can be consumed by log-analysis tools and fail2ban filters. Query strings are omitted to avoid recording invite or password-reset tokens.

The standard container stores the log at `/data/access.log` alongside other persistent application data. ComicHero appends to the file but does not rotate it; configure the host's log-rotation tooling according to your retention requirements.

## Project stack

- Go, chi, Huma, sqlx, SQLite, and goose
- Vue 3, Vue Router, Vite, and a generated service worker
- Docker, Docker Compose, and standalone release binaries

## Community and support

Questions, feedback, and reading-order discussion are welcome in the [ComicHero Discord community](https://discord.gg/GebUwAVP). For reproducible bugs and feature requests, open a [GitHub issue](https://github.com/Lofter1/ComicHero/issues).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, testing, and pull-request guidelines.

## Security

Please follow [SECURITY.md](SECURITY.md) when reporting a vulnerability.

## License

ComicHero is available under the [MIT License](LICENSE).
