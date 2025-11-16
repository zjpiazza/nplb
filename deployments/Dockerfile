# Use a multi-stage build to create a small final image.
# Stage 1: Build the Go applications
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy the Go modules files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the API server, worker, and CLI
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o /cli ./cmd/cli

# Stage 2: Create the final image
FROM alpine:latest

# Install any necessary runtime dependencies (e.g., ca-certificates, gpg)
RUN apk --no-cache add ca-certificates gpg

# Copy the built binaries from the builder stage
COPY --from=builder /api /usr/local/bin/nplb-api
COPY --from=builder /worker /usr/local/bin/nplb-worker
COPY --from=builder /cli /usr/local/bin/nplb-cli

# Expose the port the API server runs on
EXPOSE 8080

# The default command can be to run the API server.
# The worker can be run by overriding the command in docker-compose.yml.
CMD ["nplb-api"]
