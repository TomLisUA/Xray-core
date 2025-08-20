# tun-simpl - Lightweight VPN Client

Uproszczona wersja klienta VPN z minimalną funkcjonalnością (~100 linii kodu).

## Funkcjonalność

- **TUN Interface**: Tworzy interfejs `tun-simpl` z adresem `10.50.0.2/24`
- **VPS Tunnel**: Przekazuje ruch z TUN do serwera VPS
- **HTTP Proxy**: Proxy na porcie 8080 dla interfejsu web modemu
- **Raw Socket**: Nasłuchuje pakietów z VPS i przekazuje do WAN

## Kompilacja

```bash
chmod +x build_simpl.sh
./build_simpl.sh
```

## Użycie

```bash
sudo ./tun-simpl <vps_address:port>
```

Przykład:
```bash
sudo ./tun-simpl 1.2.3.4:443
```

## Architektura

```
[TUN 10.50.0.2] ←→ [VPS Server] ←→ [Internet]
       ↓
[HTTP Proxy :8080] → [Modem 192.168.24.1:80]
```

## Routing Logic

1. **TUN → VPS**: Cały ruch z interfejsu TUN jest przekazywany do serwera VPS
2. **VPS → WAN**: Pakiety z VPS są przekazywane do interfejsu WAN
3. **HTTP → Modem**: Żądania HTTP są proxy do interfejsu web modemu

## Konfiguracja

Edytuj stałe w `tun-simpl.go`:

```go
const (
    tunAddr    = "10.50.0.2/24"        // Adres interfejsu TUN
    modemAddr  = "192.168.24.1:80"     // Adres interfejsu web modemu
    proxyPort  = ":8080"               // Port HTTP proxy
)
```

## Wymagania

- Linux/macOS
- Uprawnienia root (sudo)
- CGO_ENABLED=1

## Zalety

- **Mały rozmiar**: ~100 linii kodu vs pełny XRay
- **Szybka kompilacja**: Brak ciężkich dependencies
- **Proste debugowanie**: Minimalna funkcjonalność
- **Niskie zużycie pamięci**: Brak niepotrzebnych modułów