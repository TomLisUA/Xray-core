#!/bin/bash

set -e

echo "Building tun-l3 (Full L3 invisibility)..."

export CGO_ENABLED=1

# Add WebSocket dependency
go mod tidy

# Build with minimal size
go build -ldflags "-s -w" -trimpath -o tun-l3 tun-l3.go

SIZE=$(du -h tun-l3 | cut -f1)
echo "tun-l3 binary created: tun-l3 ($SIZE)"

echo ""
echo "Full L3 tunnel with VLESS+WS/TLS ready!"
echo "Usage: sudo ./tun-l3 <vps.domain.com:443>"
echo ""
echo "Features:"
echo "✅ Raw IP packet forwarding (QUIC/HTTP3 preserved)"
echo "✅ VLESS+WebSocket/TLS on port 443"
echo "✅ Complete fingerprint invisibility"
echo "✅ Instagram/Facebook bypass"