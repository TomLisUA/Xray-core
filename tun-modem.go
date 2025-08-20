package main

import (
	"context"
	"crypto/tls"
	_ "embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/goxray/core/network/route"
	"github.com/goxray/core/network/tun"
)

//go:embed tun.ko
var tunModule []byte

const (
	tunAddr = "10.50.0.2/24"
	vpsAddr = "10.50.0.1"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <vps_domain:443>\n", os.Args[0])
		os.Exit(1)
	}

	vpsTarget := os.Args[1]
	
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup TUN interface with embedded module
	tunIface, err := setupTUN()
	if err != nil {
		log.Fatal("TUN setup failed:", err)
	}
	defer tunIface.Close()

	// Setup routing
	if err := setupRouting(vpsTarget); err != nil {
		log.Fatal("Routing setup failed:", err)
	}

	// Start L3 tunnel
	go startL3Tunnel(ctx, tunIface, vpsTarget)

	log.Printf("L3 tunnel active: %s ⇄ %s", tunAddr, vpsTarget)
	<-sigterm
	cancel()
}

func setupTUNModule() error {
	// Create /dev/net directory
	if err := os.MkdirAll("/dev/net", 0755); err != nil {
		return err
	}

	// Create TUN device node
	dev := (10 << 8) | 200 // Major 10, Minor 200
	if err := syscall.Mknod("/dev/net/tun", syscall.S_IFCHR|0666, dev); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}

	// Write embedded tun.ko to temp file
	tmpFile := "/tmp/tun.ko"
	if err := os.WriteFile(tmpFile, tunModule, 0644); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	// Load TUN module
	cmd := exec.Command("insmod", tmpFile)
	if err := cmd.Run(); err != nil {
		// Try modprobe as fallback
		cmd = exec.Command("modprobe", "tun")
		cmd.Run() // Ignore error - module might already be loaded
	}

	return nil
}

func setupTUN() (*tun.Interface, error) {
	// Setup TUN module and device node first
	if err := setupTUNModule(); err != nil {
		log.Printf("TUN module setup warning: %v", err)
	}

	iface, err := tun.New("tun-l3", 1500)
	if err != nil {
		return nil, err
	}

	ip, ipnet, err := net.ParseCIDR(tunAddr)
	if err != nil {
		return nil, err
	}

	if err := iface.Up(&net.IPNet{IP: ip, Mask: ipnet.Mask}, ip); err != nil {
		return nil, err
	}

	return iface, nil
}

func setupRouting(vpsHost string) error {
	r, err := route.New()
	if err != nil {
		return err
	}

	// Route all traffic through TUN except VPS
	vpsIP, err := net.ResolveIPAddr("ip", vpsHost[:len(vpsHost)-4]) // Remove :443
	if err != nil {
		return err
	}

	// Exception for VPS IP
	gw, err := getDefaultGateway()
	if err != nil {
		return err
	}

	err = r.Add(route.Opts{
		Gateway: gw,
		Routes:  []*route.Addr{route.MustParseAddr(vpsIP.String() + "/32")},
	})
	if err != nil {
		return err
	}

	// Route everything else through TUN
	return r.Add(route.Opts{
		IfName: "tun-l3",
		Routes: []*route.Addr{
			route.MustParseAddr("0.0.0.0/1"),
			route.MustParseAddr("128.0.0.0/1"),
		},
	})
}

func getDefaultGateway() (net.IP, error) {
	// Simplified - get from route table
	return net.IPv4(192, 168, 1, 1), nil // Replace with actual gateway detection
}

func startL3Tunnel(ctx context.Context, tunIface *tun.Interface, vpsTarget string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := dialVLESSWS(vpsTarget)
			if err != nil {
				log.Printf("VLESS connection failed: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("VLESS+WS connected to %s", vpsTarget)
			
			// Bidirectional raw packet forwarding
			go forwardTUNtoVLESS(ctx, tunIface, conn)
			forwardVLESStoTUN(ctx, conn, tunIface)
			
			conn.Close()
			log.Println("VLESS connection closed, reconnecting...")
			time.Sleep(1 * time.Second)
		}
	}
}

func dialVLESSWS(target string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ServerName: target[:len(target)-4], // Remove :443
		},
		HandshakeTimeout: 10 * time.Second,
	}

	// VLESS WebSocket handshake
	headers := http.Header{}
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	wsURL := fmt.Sprintf("wss://%s/vless", target)
	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}

	// Send VLESS handshake
	vlessHandshake := buildVLESSHandshake()
	if err := conn.WriteMessage(websocket.BinaryMessage, vlessHandshake); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func buildVLESSHandshake() []byte {
	// Minimal VLESS handshake for TUN mode
	handshake := make([]byte, 16+1+16+1+2+1+1)
	// UUID (16 bytes) + Version (1) + Encryption (16) + Reserved (1) + Command (2) + Port (1) + Address Type (1)
	handshake[16] = 0x00 // Version
	handshake[33] = 0x00 // Reserved  
	handshake[34] = 0x03 // Command: TUN mode
	handshake[35] = 0x00 // Port high
	handshake[36] = 0x00 // Port low
	handshake[37] = 0x01 // Address type: IPv4
	return handshake
}

func forwardTUNtoVLESS(ctx context.Context, tunIface *tun.Interface, conn *websocket.Conn) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := tunIface.Read(buf)
			if err != nil {
				return
			}

			// Forward raw IP packet through VLESS WebSocket
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}
}

func forwardVLESStoTUN(ctx context.Context, conn *websocket.Conn, tunIface *tun.Interface) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, packet, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Write raw IP packet back to TUN (preserves QUIC/HTTP3)
			if _, err := tunIface.Write(packet); err != nil {
				return
			}
		}
	}
}