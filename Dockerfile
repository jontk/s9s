# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o s9s ./cmd/s9s

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 1000 s9s

# Copy binary from builder
COPY --from=builder /app/s9s /usr/local/bin/s9s

# Create config directory
RUN mkdir -p /home/s9s/.s9s && chown -R s9s:s9s /home/s9s

# Switch to non-root user
USER s9s

# Set working directory
WORKDIR /home/s9s

# Default command
ENTRYPOINT ["s9s"]