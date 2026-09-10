# Multi-stage build: frontend -> Go binary.

# --- Stage 1: build React frontend ---
FROM node:20-alpine AS frontend
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# --- Stage 2: build Go binary ---
FROM golang:1.22-alpine AS backend
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
# TODO: embed frontend dist via go:embed once wired up.
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

# --- Stage 3: runtime ---
FROM alpine:3.20
RUN adduser -D app
USER app
WORKDIR /app
COPY --from=backend /bin/server /app/server
COPY migrations/ /app/migrations/

ENV HTTP_PORT=8080 \
    DB_PATH=/data/autocheck.db \
    IMPORT_DIR=/data/imports \
    IMPORT_INTERVAL=300s

EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/app/server"]
