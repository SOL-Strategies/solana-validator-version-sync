#!/bin/bash

# Test script to simulate the release process locally
# This script uses the same Make target as GitHub Actions.

set -e

echo "🧪 Testing release process locally..."

# Build release archives with a test version
make build-all APP_VERSION="1.0.0-test"

echo ""
echo "✅ Release test complete!"
echo "📁 Files are in the bin/ directory"
echo "🧪 Test a binary: gunzip ./bin/solana-validator-version-sync-1.0.0-test-linux-amd64.gz && ./bin/solana-validator-version-sync-1.0.0-test-linux-amd64 --version"
