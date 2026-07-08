.PHONY: dev dev-backend dev-ui test test-backend test-ui build build-backend build-ui install-ui build-standalone docker-build docker-run compose-up compose-down

GOCACHE ?= $(CURDIR)/tmp/go-build
export GOCACHE

BACKEND_URL ?= http://localhost:8080/api/docs
UI_URL ?= http://localhost:5173
OPEN_BROWSER ?= 1
OPEN_CMD ?= open
IMAGE ?= comichero:latest
VITE_API_BASE ?= /api
export VITE_API_BASE
DIST_DIR ?= dist
VERSION ?= dev
BINARY_NAME ?= comichero

dev:
	@set -e; \
	(cd backend && air -color never -screen.clear_on_rebuild false) 2>&1 | awk '{ print "[backend] " $$0; fflush() }' & backend_pid=$$!; \
	(npm --prefix ui run dev) 2>&1 | awk '{ print "[ui] " $$0; fflush() }' & ui_pid=$$!; \
	if [ "$(OPEN_BROWSER)" = "1" ]; then \
		(sleep 2; $(OPEN_CMD) "$(BACKEND_URL)"; $(OPEN_CMD) "$(UI_URL)") & \
	fi; \
	trap 'kill $$backend_pid $$ui_pid 2>/dev/null' INT TERM EXIT; \
	wait $$backend_pid $$ui_pid

dev-backend:
	cd backend && air -screen.clear_on_rebuild false

dev-ui:
	npm --prefix ui run dev

test: test-backend test-ui

test-backend:
	cd backend && go test ./...

test-ui:
	npm --prefix ui run build

build: build-backend build-ui

build-backend:
	cd backend && go build ./...

build-ui:
	npm --prefix ui run build

install-ui:
	npm --prefix ui install

build-standalone: build-ui
	rm -rf backend/internal/static/dist
	mkdir -p backend/internal/static/dist
	cp -r ui/dist/. backend/internal/static/dist/
	mkdir -p $(DIST_DIR)
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$(VERSION)" -o ../$(DIST_DIR)/$(BINARY_NAME) .
	@echo "==> built $(DIST_DIR)/$(BINARY_NAME) (standalone binary, frontend embedded, no other files needed)"

docker-build:
	docker build --build-arg VITE_API_BASE=$(VITE_API_BASE) -t $(IMAGE) .

docker-run:
	docker run --rm -p 8080:8080 -v comichero-data:/data $(IMAGE)

compose-up:
	docker compose up --build

compose-down:
	docker compose down

lint: lint-ui

lint-ui:
	npm --prefix ui run lint
	npm --prefix ui run format