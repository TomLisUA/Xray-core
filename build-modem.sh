#!/bin/bash

set -e

echo "Building tun-modem (ARMv7 with embedded TUN module)..."

# Build ARMv7 static binary with embedded tun.ko
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=arm
export GOARM=7

go build -ldflags="-s -w" -o tun-modem tun-modem.go

SIZE=$(du -h tun-modem | cut -f1)
echo "Modem-ready binary created: tun-modem ($SIZE)"

echo ""
echo "✅ Features included:"
echo "  - Embedded tun.ko module"
echo "  - Automatic /dev/net/tun creation"
echo "  - Static ARMv7 binary"
echo "  - Full L3 VLESS+WS/TLS tunnel"
echo ""
echo "Deploy to modem:"
echo "  scp tun-modem root@192.168.24.1:/tmp/"
echo "  ssh root@192.168.24.1 'chmod +x /tmp/tun-modem'"
echo "  ssh root@192.168.24.1 '/tmp/tun-modem vps.domain.com:443'"