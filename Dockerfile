# ── Stage 1: build the React dashboard ────────────────────────────────────────
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ .
RUN npm run build

# ── Stage 2: build the Go binary (with embedded UI) ───────────────────────────
FROM golang:1.25-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Place the Vite build output where the embed directive points.
COPY --from=web-builder /app/web/dist ./cmd/server/ui/
RUN go build -tags ui -o /mission-control ./cmd/server

# ── Stage 3: minimal runtime image ────────────────────────────────────────────
FROM alpine:3.21
COPY --from=go-builder /mission-control /usr/local/bin/mission-control
EXPOSE 5040
ENTRYPOINT ["mission-control"]
