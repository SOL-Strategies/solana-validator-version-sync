FROM golang:1.27.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata bash make

# Set working directory
WORKDIR /app

COPY . .

# Download dependencies without changing the module files.
RUN go mod download

ARG APP_VERSION=dev
RUN APP_VERSION="$APP_VERSION" sh scripts/prepare-version.sh \
    && CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags='-s -w' \
       -o /solana-validator-version-sync ./cmd/solana-validator-version-sync

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /solana-validator-version-sync /usr/local/bin/solana-validator-version-sync
ENTRYPOINT ["solana-validator-version-sync"]
