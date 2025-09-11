FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o solana-validator-version-sync ./cmd/solana-validator-version-sync

# Create final minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/solana-validator-version-sync .

# Create config directory (if needed)
# RUN mkdir -p /root/solana-validator-version-sync

# Expose port (if needed for health checks)
EXPOSE 8080

# Run the application
CMD ["./solana-validator-version-sync"]
