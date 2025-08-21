#!/bin/bash

set -e

echo "Cross-compiling for E3372H ARMv7..."

# Install ARM cross-compiler
sudo apt-get update
sudo apt-get install -y gcc-arm-linux-gnueabihf

# Set cross-compilation environment
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=arm
export GOARM=7
export CC=arm-linux-gnueabihf-gcc

# Build ARMv7 binary
go build -ldflags="-s -w" -o tun-e3372h-arm tun-optimized.go

# Verify it's ARM
file tun-e3372h-arm

SIZE=$(du -h tun-e3372h-arm | cut -f1)
echo "E3372H ARMv7 binary ready: tun-e3372h-arm ($SIZE)"

echo ""
echo "Deploy: scp tun-e3372h-arm root@192.168.24.1:/tmp/"