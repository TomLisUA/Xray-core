# VPS Configuration for L3 Tunnel

## XRay Server Config (vps.json)

```json
{
  "inbounds": [{
    "port": 443,
    "protocol": "vless",
    "settings": {
      "clients": [{
        "id": "your-uuid-here"
      }]
    },
    "streamSettings": {
      "network": "ws",
      "security": "tls",
      "tlsSettings": {
        "certificates": [{
          "certificateFile": "/path/to/cert.pem",
          "keyFile": "/path/to/key.pem"
        }]
      },
      "wsSettings": {
        "path": "/vless"
      }
    }
  }],
  "outbounds": [{
    "protocol": "freedom",
    "tag": "direct"
  }],
  "routing": {
    "rules": [{
      "type": "field",
      "outboundTag": "direct"
    }]
  }
}
```

## TUN Interface Setup on VPS

```bash
#!/bin/bash
# Setup TUN interface on VPS for L3 forwarding

# Create TUN interface
ip tuntap add mode tun dev tun0
ip addr add 10.50.0.1/24 dev tun0
ip link set tun0 up

# Enable IP forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward

# Route traffic from TUN to internet
iptables -t nat -A POSTROUTING -s 10.50.0.0/24 -o eth0 -j MASQUERADE
iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT
iptables -A FORWARD -i eth0 -o tun0 -m state --state RELATED,ESTABLISHED -j ACCEPT

# Set default route for TUN traffic
ip route add 10.50.0.2/32 dev tun0
```

## Start Services

```bash
# Start XRay server
./xray -config vps.json

# Setup TUN interface
sudo ./setup-tun.sh
```

## Architecture Flow

```
[App on VPS] → QUIC/HTTP3 → [tun0: 10.50.0.1] → [XRay VLESS] → [WS/TLS:443] → [Modem] → [LTE] → [Instagram]
```

**Result**: Instagram sees original QUIC fingerprint from modem's LTE IP = Complete invisibility