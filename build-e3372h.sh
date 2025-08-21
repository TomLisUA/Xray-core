#!/bin/bash

set -e

echo "Building tun-optimized for Huawei E3372H..."

# Use buildx for proper ARM cross-compilation
docker buildx build --platform linux/arm/v7 -t tun-builder . -f - <<EOF
FROM --platform=linux/arm/v7 golang:1.21-alpine
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm GOARM=7 \
    go build -ldflags="-s -w -extldflags=-static" \
    -gcflags="-l=4" -o tun-e3372h tun-optimized.go
EOF

# Extract binary from container
docker create --name temp-container tun-builder
docker cp temp-container:/app/tun-e3372h ./tun-e3372h
docker rm temp-container

SIZE=$(du -h tun-e3372h | cut -f1)
echo "E3372H ARMv7 binary: tun-e3372h ($SIZE)"

echo ""
echo "🎯 E3372H Ready:"
echo "  ✅ ARMv7 static binary"
echo "  ✅ Embedded tun.ko module"
echo "  ✅ Memory optimized for 41MB RAM"
echo ""
echo "Deploy to E3372H:"
echo "  scp tun-e3372h root@192.168.24.1:/tmp/"
echo "  ssh root@192.168.24.1 '/tmp/tun-e3372h vps.domain.com:443'"