#!/bin/sh
set -eu

GO=${GO:-go}
APP_NAME=${APP_NAME:-solana-validator-version-sync}
APP_VERSION=${APP_VERSION:-dev}
APP_VERSION=${APP_VERSION#v}
APP_VERSION=$(printf '%s' "$APP_VERSION" | tr '/' '-')
BUILD_DIR=${BUILD_DIR:-bin}
BUILD_OS_ARCH_LIST=${BUILD_OS_ARCH_LIST:-linux-amd64}
PACKAGE=${PACKAGE:-false}

sh scripts/prepare-version.sh
mkdir -p "$BUILD_DIR"

for platform in $BUILD_OS_ARCH_LIST; do
    case "$platform" in
        linux-amd64|linux-arm64) ;;
        *) printf 'Unsupported build platform: %s\n' "$platform" >&2; exit 1 ;;
    esac
    output="$BUILD_DIR/$APP_NAME-$APP_VERSION-$platform"
    printf 'Building %s with %s\n' "$output" "$($GO env GOVERSION)"
    CGO_ENABLED=0 GOOS=${platform%-*} GOARCH=${platform#*-} \
        "$GO" build -mod=readonly -trimpath -ldflags='-s -w' \
        -o "$output" ./cmd/solana-validator-version-sync

    if [ "$PACKAGE" = true ]; then
        gzip -n -f "$output"
        (
            cd "$BUILD_DIR"
            archive="$APP_NAME-$APP_VERSION-$platform.gz"
            if command -v sha256sum >/dev/null 2>&1; then
                sha256sum "$archive" > "$archive.sha256"
            else
                shasum -a 256 "$archive" > "$archive.sha256"
            fi
        )
    fi
done
