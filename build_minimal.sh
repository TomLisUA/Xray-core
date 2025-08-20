#!/bin/bash

# Minimal build script for Go XRay VPN client
# This script builds a minimal version excluding unused protocols and features

set -e

echo "Building minimal Go XRay VPN client..."

# Build flags for minimal binary
BUILD_FLAGS="-ldflags=-s -w -trimpath"
TAGS="minimal"

# Set CGO_ENABLED=1 as required by the project
export CGO_ENABLED=1

# Build the minimal binary
echo "Compiling with minimal feature set..."
go build $BUILD_FLAGS -tags $TAGS -o goxray_minimal .

# Get binary size
SIZE=$(du -h goxray_minimal | cut -f1)
echo "Minimal binary created: goxray_minimal ($SIZE)"

echo "Build completed successfully!"
echo ""
echo "Usage: sudo ./goxray_minimal <vless://your-server-link>"