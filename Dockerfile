FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata bash make

# Set working directory
WORKDIR /app

COPY . .

# Download dependencies
RUN go mod tidy
