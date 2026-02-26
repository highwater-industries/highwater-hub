## ── Stage 1: Build the SvelteKit frontend ──
FROM node:22-alpine AS frontend

WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

## ── Stage 2: Build the Go binary (with embedded frontend) ──
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .
# Copy the frontend build output so go:embed can pick it up
COPY --from=frontend /app/web/build ./internal/frontend/dist
RUN go build -o server ./cmd/server

## ── Stage 3: Minimal runtime ──
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 3141

CMD ["./server"]
