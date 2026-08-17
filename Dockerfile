# syntax=docker/dockerfile:1.7
# Multi-stage Dockerfile for design-review platform.

FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS backend
WORKDIR /app
RUN apk add --no-cache git
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /bin/server /app/server
COPY migrations /app/migrations
COPY api /app/api
ENV HTTP_ADDR=:8080 \
    RUNTIME_MODE=memory \
    STORAGE_BASE_DIR=/app/var/attachments \
    LOG_LEVEL=info
EXPOSE 8080
VOLUME ["/app/var"]
ENTRYPOINT ["/app/server"]
