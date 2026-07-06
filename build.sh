#!/usr/bin/env bash
# Builds a single, self-contained ComicHero binary: compiles the frontend,
# copies it into the backend's embed package, then compiles a static Go
# binary. The result (dist/comichero) needs no other files at runtime
# besides a writable directory for the SQLite DB and cover cache.
set -euo pipefail

cd "$(dirname "$0")"

echo "==> building frontend"
(cd ui && npm ci && npm run build)

echo "==> embedding frontend into backend"
rm -rf backend/internal/static/dist
mkdir -p backend/internal/static/dist
cp -r ui/dist/. backend/internal/static/dist/

echo "==> building binary"
mkdir -p dist
(cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../dist/comichero .)

echo "==> done: dist/comichero"
echo "    run with: DB_PATH=./data/comicorder.db COVER_CACHE_DIR=./data/covers ./dist/comichero"
