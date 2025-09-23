# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cnet-agent main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 cnet && \
    adduser -D -s /bin/sh -u 1001 -G cnet cnet

# Create directories
RUN mkdir -p /app /var/lib/cnet /tmp/cnet && \
    chown -R cnet:cnet /app /var/lib/cnet /tmp/cnet

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/cnet-agent .

# Copy configuration
COPY config.yaml .

# Switch to non-root user
USER cnet

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./cnet-agent", "-config", "config.yaml"]
