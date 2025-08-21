#!/bin/bash

set -e

echo "Building tun-optimized for Huawei E3372H..."

# E3372H specific optimizations
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=arm
export GOARM=7

# Aggressive size optimization for 41MB RAM device
LDFLAGS="-s -w -extldflags '-static'"
GCFLAGS="-l=4"  # Aggressive inlining

go build -ldflags="$LDFLAGS" -gcflags="$GCFLAGS" -o tun-e3372h tun-optimized.go

# Strip additional symbols
strip --strip-all tun-e3372h 2>/dev/null || true

SIZE=$(du -h tun-e3372h | cut -f1)
echo "E3372H optimized binary: tun-e3372h ($SIZE)"

echo ""
echo "🎯 E3372H Optimizations:"
echo "  ✅ OOM killer protection (-1000)"
echo "  ✅ High process priority (-20)"
echo "  ✅ Small buffers (2KB vs 64KB)"
echo "  ✅ Limited connections (8 vs 32)"
echo "  ✅ Single CPU core (GOMAXPROCS=1)"
echo "  ✅ Embedded tun.ko module"
echo ""
echo "Deploy: scp tun-e3372h root@192.168.24.1:/tmp/"