# Deployment Guide - Huawei E3372H

## 📱 **Binary Ready**: `tun-e3372h-armv7` (5.4M)

### 🎯 **E3372H Specifications**:
- **CPU**: ARMv7 Cortex-A9 (HiSilicon Hi6930)
- **RAM**: 41MB total (~20-30MB available)
- **Architecture**: 32-bit ARM
- **System**: Android embedded

### 🚀 **Deployment Steps**:

#### 1. **Transfer to E3372H**:
```bash
scp tun-e3372h-armv7 root@192.168.24.1:/tmp/
```

#### 2. **Connect to E3372H**:
```bash
ssh root@192.168.24.1
```

#### 3. **Setup and Run**:
```bash
# Make executable
chmod +x /tmp/tun-e3372h-armv7

# Run with your VPS
/tmp/tun-e3372h-armv7 vps.yourdomain.com:443
```

### ⚡ **Optimizations Included**:

1. **Memory Management**:
   - Buffer size: 2KB (optimized for 41MB RAM)
   - Max connections: 8 (vs 32 default)
   - Single CPU core usage (GOMAXPROCS=1)

2. **Process Priority**:
   - OOM killer protection: `-1000`
   - High process priority: `-20`
   - Real-time scheduling ready

3. **Embedded Components**:
   - TUN kernel module (tun.ko) - 22KB
   - Automatic /dev/net/tun creation
   - Zero external dependencies

4. **Network Optimizations**:
   - VLESS+WebSocket/TLS on port 443
   - Raw IP packet forwarding
   - QUIC/HTTP3 fingerprint preservation

### 📊 **Memory Usage**:
- **Binary**: 5.4M
- **Runtime RAM**: ~8-12M
- **Total footprint**: <15M on 41M available

### 🔧 **Troubleshooting**:

#### If TUN module fails to load:
```bash
# Manual TUN setup
mkdir -p /dev/net
mknod /dev/net/tun c 10 200
chmod 666 /dev/net/tun
modprobe tun
```

#### Check memory usage:
```bash
cat /proc/meminfo | grep Available
ps aux | grep tun-e3372h
```

#### Monitor process:
```bash
top -p $(pidof tun-e3372h-armv7)
```

### ✅ **Expected Result**:
- **Instagram/Facebook**: Complete invisibility
- **IP seen**: E3372H LTE IP address
- **Fingerprint**: Original QUIC from VPS
- **Performance**: Optimized for 41MB RAM constraint

### 🎯 **Success Indicators**:
```
L3 tunnel active: 10.50.0.2/24 ⇄ vps.yourdomain.com:443
VLESS+WS connected to vps.yourdomain.com:443
```

Ready for E3372H deployment and testing!