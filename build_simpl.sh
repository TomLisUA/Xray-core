#!/bin/bash

# Build script for tun-simpl - lightweight VPN client
set -e

echo "Building tun-simpl (lightweight version)..."

# Set CGO_ENABLED=1 for TUN interface support
export CGO_ENABLED=1

# Build the simplified binary
echo "Compiling tun-simpl..."
go build -ldflags "-s -w" -trimpath -o tun-simpl tun-simpl.go

# Get binary size
SIZE=$(du -h tun-simpl | cut -f1)
echo "tun-simpl binary created: tun-simpl ($SIZE)"

echo "Build completed successfully!"
echo ""
echo "Usage: sudo ./tun-simpl <vps_address:port>"
echo "Example: sudo ./tun-simpl 1.2.3.4:443"