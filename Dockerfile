# syntax=docker/dockerfile:1

FROM node:26-alpine AS ui-build
WORKDIR /src/ui

COPY ui/package*.json ./
RUN npm ci

COPY ui/ ./
ARG VITE_API_BASE="/api"
ENV VITE_API_BASE=$VITE_API_BASE
RUN npm run build

FROM golang:1.26.4-alpine AS backend-build
WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
COPY --from=ui-build /src/ui/dist ./internal/static/dist
ARG VERSION="dev"
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /out/comichero .

FROM alpine:3.22
RUN adduser -D -H comichero \
    && mkdir -p /data \
    && chown -R comichero /data

WORKDIR /app
COPY --from=backend-build /out/comichero /app/comichero

ENV PORT=8080
ENV DB_PATH=/data/comicorder.db
ENV COVER_CACHE_DIR=/data/covers

EXPOSE 8080
VOLUME ["/data"]

USER comichero
CMD ["/app/comichero"]
