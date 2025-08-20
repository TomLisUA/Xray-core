#!/bin/bash

set -e

echo "Building tun-l3 for ARMv7 (static binary)..."

# Build ARMv7 static binary using Docker
docker run --rm --platform=linux/arm/v7 \
  -v "$PWD":/app -w /app messense/musl-cross:armv7 \
  sh -c 'CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 \
         CC=arm-linux-musleabihf-gcc \
         go build -ldflags="-s -w" -o tun-l3-armv7 tun-l3.go'

SIZE=$(du -h tun-l3-armv7 | cut -f1)
echo "ARMv7 binary created: tun-l3-armv7 ($SIZE)"

echo ""
echo "Static ARMv7 binary ready for modem deployment!"
echo "Transfer to modem: scp tun-l3-armv7 root@192.168.24.1:/tmp/"
echo "Run on modem: sudo ./tun-l3-armv7 vps.domain.com:443"