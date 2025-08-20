# Architektura tunelowania L3 dla pełnej niewidzialności

## Obecny problem z tun-simpl

**Brakuje**: Prawdziwe tunelowanie L3 z zachowaniem oryginalnych pakietów IP/UDP.

## Wymagana architektura

```
[VPS: 10.50.0.1] ⇄ (VLESS+WS/TLS:443) ⇄ [Modem: 10.50.0.2] → LTE → Internet
```

### Kluczowe elementy dla niewidzialności:

#### 1. **VLESS+WebSocket/TLS na porcie 443**
```go
// Modem inicjuje połączenie wyglądające jak HTTPS
conn := dialVLESSTLS("vps.example.com:443")
```

#### 2. **Tunelowanie raw IP packets (L3)**
```go
// Pakiety IP 1:1 bez modyfikacji
tunPacket := readFromTUN()  // Raw IP packet
vlessConn.Write(tunPacket)  // Przesyła bez zmian
```

#### 3. **Routing na VPS**
```bash
# VPS konfiguracja
ip tuntap add mode tun dev tun0
ip addr add 10.50.0.1/24 dev tun0
ip route add default via 10.50.0.2 dev tun0
```

## Co daje pełną niewidzialność:

### ✅ **QUIC/HTTP3 fingerprint preservation**
- Aplikacja na VPS generuje oryginalny QUIC handshake
- Pakiet przechodzi przez tunel bez modyfikacji
- Modem wysyła identyczny pakiet przez LTE
- **Instagram/Facebook widzą**: oryginalny fingerprint aplikacji

### ✅ **IP masking**
- Zewnętrzny IP = IP modemu LTE
- Geolokalizacja = lokalizacja modemu
- CGNAT nie widzi tunelowania (wygląda jak HTTPS)

### ✅ **Protocol transparency**
```
VPS App → QUIC packet → TUN → VLESS → WS/TLS → Modem → LTE → Instagram
```

## Brakujące komponenty w tun-simpl:

### 1. **Prawdziwy VLESS protocol**
```go
func dialVLESS(addr string) (net.Conn, error) {
    // Implementacja VLESS handshake
    // WebSocket upgrade
    // TLS wrapping
}
```

### 2. **Raw packet handling**
```go
// Obecnie: TCP stream
// Potrzebne: Raw IP packets
func forwardRawIP(packet []byte) {
    // Bez parsowania/modyfikacji
    vlessConn.Write(packet)
}
```

### 3. **Proper routing setup**
```go
// Dodanie tras dla pełnego tunelowania
func setupRoutes() {
    // Route all traffic through TUN
    // Exception for VPS IP
}
```

## Rezultat:

**Z obecnym tun-simpl**: Instagram może wykryć proxy/VPN  
**Z pełnym L3 tunelem**: Instagram widzi normalną aplikację z IP modemu

### Fingerprint comparison:
```
Bez tunelu L3:  [Proxy headers] + [Modified packets] = 🚫 Detected
Z tunelem L3:   [Original QUIC] + [Modem IP] = ✅ Invisible
```