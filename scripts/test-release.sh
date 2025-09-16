#!/bin/bash

# Test script to simulate the release process locally
# This script uses the build-release.sh script to test the complete process

set -e

echo "🧪 Testing release process locally..."

# Use the build script with a test version
./scripts/build-release.sh "1.0.0-test"

echo ""
echo "✅ Release test complete!"
echo "📁 Files are in the bin/ directory"
echo "🧪 Test a binary: gunzip ./bin/solana-validator-version-sync-1.0.0-test-linux-amd64.gz && ./bin/solana-validator-version-sync-1.0.0-test-linux-amd64 --version"
