# Build stage
FROM golang:1.25-alpine AS builder

# Install git (required for some Go modules)
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bloomdb .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and postgresql client
RUN apk --no-cache add ca-certificates postgresql-client

# Create app user
RUN addgroup -g 1001 -S bloomdb && \
    adduser -u 1001 -S bloomdb -G bloomdb

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/bloomdb .

# Copy migrations folder from builder stage
COPY --from=builder /app/migrations ./migrations

# Set ownership of migrations directory
RUN chown -R bloomdb:bloomdb /app/migrations

# Set default environment variables
ENV BLOOMDB_PATH=/app/migrations

# Change to app user
USER bloomdb

# Default command (can be overridden)
CMD ["./bloomdb"]