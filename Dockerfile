# syntax=docker/dockerfile:1

FROM node:26-alpine AS ui-build
WORKDIR /src/ui

COPY ui/package*.json ./
RUN npm ci

COPY ui/ ./
ARG VITE_API_BASE="/api"
ENV VITE_API_BASE=$VITE_API_BASE
RUN npm run build

FROM golang:1.26-alpine AS backend-build
WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/comichero .

FROM alpine:3.22
RUN adduser -D -H comichero \
    && mkdir -p /app/public /data \
    && chown -R comichero /app /data

WORKDIR /app
COPY --from=backend-build /out/comichero /app/comichero
COPY --from=ui-build /src/ui/dist /app/public

ENV PORT=8080
ENV DB_PATH=/data/comicorder.db
ENV COVER_CACHE_DIR=/data/covers
ENV STATIC_DIR=/app/public

EXPOSE 8080
VOLUME ["/data"]

USER comichero
CMD ["/app/comichero"]
