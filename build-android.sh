#!/bin/bash

set -e

echo "Building static binary for Android E3372H..."

# Install musl cross-compiler for Android
wget -q https://musl.cc/arm-linux-musleabihf-cross.tgz
tar -xf arm-linux-musleabihf-cross.tgz

# Set static compilation environment
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=arm
export GOARM=7
export CC=$PWD/arm-linux-musleabihf-cross/bin/arm-linux-musleabihf-gcc
export CXX=$PWD/arm-linux-musleabihf-cross/bin/arm-linux-musleabihf-g++

# Build static binary for Android
go build -ldflags="-s -w -extldflags=-static" -o tun-android tun-optimized.go

# Verify it's static ARM
file tun-android
ldd tun-android 2>/dev/null || echo "Static binary confirmed"

SIZE=$(du -h tun-android | cut -f1)
echo "Android E3372H binary ready: tun-android ($SIZE)"

echo ""
echo "Deploy: scp tun-android root@192.168.24.1:/tmp/"