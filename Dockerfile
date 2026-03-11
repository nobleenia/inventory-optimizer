# ───────────────────────────────────────────────────────────────
# Inventory Optimizer — Multi-stage Docker build
#
# Produces a ~25 MB scratch-based image containing only the static
# Go binary + embedded web assets. No OS, no shell, no attack surface.
#
# Build:
#   docker build -t inventory-optimizer:0.4.0 .
#
# Run (web mode):
#   docker run -p 8080:8080 inventory-optimizer:0.4.0 -web
#
# Run (API mode — connect to Postgres):
#   docker run -p 8080:8080 \
#     -e DATABASE_URL=postgres://inventory:inventory@host.docker.internal:5433/inventory?sslmode=disable \
#     -e JWT_SECRET=change-me-in-production \
#     inventory-optimizer:0.4.0 -api
# ───────────────────────────────────────────────────────────────

# ── Stage 1: build the Go binary ──────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static CGO-free binary with embedded assets.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -trimpath -o /bin/inventory-optimizer ./cmd/

# ── Stage 2: minimal runtime image ────────────────────────────
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/inventory-optimizer /inventory-optimizer

EXPOSE 8080

ENTRYPOINT ["/inventory-optimizer"]
CMD ["-web", "-port", ":8080"]
