#!/bin/bash

# Build script for cross-platform binaries with compression and checksums
# Usage: ./scripts/build-release.sh <version>
# Example: ./scripts/build-release.sh 1.0.0

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

VERSION="$1"
BINARY_NAME="solana-validator-version-sync"

# Define platforms to build for
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

echo "🚀 Building $BINARY_NAME version $VERSION for all platforms..."

# Create build directory
mkdir -p bin

# Set version in version.txt file
echo "$VERSION" > cmd/version.txt

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r os arch <<< "$platform"
    output_name="${BINARY_NAME}-${VERSION}-${os}-${arch}"
    
    echo "📦 Building for $os/$arch..."
    docker run --rm -v "$(pwd)":/app -w /app golang:1.25-alpine sh -c "
        apk add --no-cache git ca-certificates &&
        go mod download &&
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -mod=mod -ldflags='-s -w' -o bin/$output_name ./cmd/solana-validator-version-sync
    "
done

echo "🗜️ Compressing binaries..."
cd bin
for binary in ${BINARY_NAME}-${VERSION}-*; do
    if [ -f "$binary" ]; then
        echo "Compressing $binary..."
        gzip "$binary"
    fi
done

echo "🔍 Generating checksums..."
for binary in ${BINARY_NAME}-${VERSION}-*.gz; do
    if [ -f "$binary" ]; then
        echo "Generating checksum for $binary..."
        sha256sum "$binary" | cut -d' ' -f1 > "${binary}.sha256"
    fi
done

echo "📋 Release contents:"
echo "Compressed binaries:"
ls -la *.gz 2>/dev/null || true
echo ""
echo "Checksum files:"
ls -la *.sha256 2>/dev/null || true

echo ""
echo "✅ Build complete!"
echo "📁 Files are in the bin/ directory"
echo "🧪 Test a binary: gunzip ./bin/${BINARY_NAME}-${VERSION}-linux-amd64.gz && ./bin/${BINARY_NAME}-${VERSION}-linux-amd64 --version"
