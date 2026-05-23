# syntax=docker/dockerfile:1.7

# ---- Stage 1: Build ----
FROM golang:1.26-alpine AS builder

# Install git for go mod (some deps may use git protocol)
RUN apk add --no-cache git

WORKDIR /app

# Cache deps separately from source for faster rebuilds.
# Changes to source code won't invalidate the dep download layer.
COPY go.mod go.sum ./
RUN go mod download

# Now copy source and build.
COPY . .

# CGO_ENABLED=0 produces a statically-linked binary that runs on scratch.
# -ldflags="-s -w" strips debug symbols, shrinking the binary.
# -trimpath removes local file paths from the binary (cleaner, more reproducible).
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /out/server \
    ./cmd/server

# ---- Stage 2: Runtime ----
FROM alpine:3.19

# Add CA certificates for HTTPS calls and tzdata for time zone support.
RUN apk add --no-cache ca-certificates tzdata

# Run as a non-root user. Hardcoding UID/GID avoids reliance on
# /etc/passwd lookups and works with read-only filesystems.
RUN addgroup -S -g 1000 app && adduser -S -u 1000 -G app app

WORKDIR /app

# Copy the binary and migrations from the builder stage.
COPY --from=builder /out/server ./server
COPY --from=builder /app/migrations ./migrations

USER app

EXPOSE 8080

# No CMD shell form — exec form means signals reach the binary directly,
# which is what makes graceful shutdown work in containers.
ENTRYPOINT ["./server"]