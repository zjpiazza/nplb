# Use a multi-stage build to create a small final image.
# Stage 1: Build the Go applications
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk --no-cache add git

# Copy the Go modules files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the API server, worker, and CLI
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /cli ./cmd/cli

# Stage 2: Create the final image
FROM alpine:latest

# Install runtime dependencies
# - ca-certificates: for HTTPS requests
# - gpg: for signing packages
# - gnupg-utils: additional GPG utilities
RUN apk --no-cache add ca-certificates gpg gnupg

# Create non-root user for security
RUN addgroup -g 1000 nplb && adduser -u 1000 -G nplb -s /bin/sh -D nplb

# Create necessary directories
RUN mkdir -p /app/keys /app/build && chown -R nplb:nplb /app

# Copy the built binaries from the builder stage
COPY --from=builder /api /usr/local/bin/nplb-api
COPY --from=builder /worker /usr/local/bin/nplb-worker
COPY --from=builder /cli /usr/local/bin/nplb-cli

# Switch to non-root user
USER nplb
WORKDIR /app

# Expose the port the API server runs on
EXPOSE 8080

# The default command is to run the API server.
# The worker can be run by overriding the command in docker-compose.yml.
CMD ["nplb-api"]
