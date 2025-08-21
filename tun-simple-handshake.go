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
	"runtime"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/goxray/core/network/tun"
)

//go:embed tun.ko
var tunModule []byte

const (
	tunAddr = "10.50.0.2/24"
	bufferSize = 2048
)

func main() {
	optimizeForAndroid()
	
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <vps_domain:8080>\n", os.Args[0])
		os.Exit(1)
	}

	vpsTarget := os.Args[1]
	
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get VPS IP and fix routing
	vpsIP, err := net.ResolveIPAddr("ip", vpsTarget[:len(vpsTarget)-5])
	if err != nil {
		log.Fatal("Failed to resolve VPS IP:", err)
	}
	log.Printf("VPS IP: %s", vpsIP.String())

	fixE3372HRouting(vpsIP.String())

	// Setup TUN interface
	tunIface, err := setupTUN()
	if err != nil {
		log.Fatal("TUN setup failed:", err)
	}
	defer tunIface.Close()

	addTUNRoutes()

	go startL3Tunnel(ctx, tunIface, vpsTarget)

	log.Printf("E3372H tunnel active: %s ⇄ %s", tunAddr, vpsTarget)
	<-sigterm
	cancel()
}

func fixE3372HRouting(vpsIP string) {
	exec.Command("ip", "route", "del", vpsIP, "via", "192.168.24.1", "dev", "br0").Run()
	exec.Command("ip", "route", "add", vpsIP, "via", "10.64.64.1", "dev", "wan0").Run()
}

func addTUNRoutes() {
	exec.Command("ip", "route", "add", "0.0.0.0/1", "dev", "tun-e3372h", "metric", "100").Run()
	exec.Command("ip", "route", "add", "128.0.0.0/1", "dev", "tun-e3372h", "metric", "100").Run()
}

func optimizeForAndroid() {
	if f, err := os.OpenFile("/proc/self/oom_score_adj", os.O_WRONLY, 0); err == nil {
		f.WriteString("-1000")
		f.Close()
	}
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, -20)
	runtime.GOMAXPROCS(1)
	runtime.GC()
}

func setupTUNModule() error {
	os.MkdirAll("/dev/net", 0755)
	
	dev := (10 << 8) | 200
	syscall.Mknod("/dev/net/tun", syscall.S_IFCHR|0666, dev)

	tmpFile := "/tmp/tun.ko"
	if err := os.WriteFile(tmpFile, tunModule, 0644); err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	exec.Command("insmod", tmpFile).Run()
	return nil
}

func setupTUN() (*tun.Interface, error) {
	setupTUNModule()

	iface, err := tun.New("tun-e3372h", 1500)
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

func startL3Tunnel(ctx context.Context, tunIface *tun.Interface, vpsTarget string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := dialVLESSWS(vpsTarget)
			if err != nil {
				log.Printf("Connection failed: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Printf("VLESS+WS connected to %s", vpsTarget)
			
			// Start packet forwarding immediately
			go forwardTUNtoVLESS(ctx, tunIface, conn)
			forwardVLESStoTUN(ctx, conn, tunIface)
			
			conn.Close()
			time.Sleep(2 * time.Second)
		}
	}
}

func dialVLESSWS(target string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	headers.Set("User-Agent", "Mozilla/5.0 (Linux; Android)")
	
	wsURL := fmt.Sprintf("ws://%s/tun", target)
	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}

	// Minimal VLESS handshake - just send UUID and start forwarding
	uuid := []byte{
		0xd4, 0x33, 0x08, 0xce, 0x0c, 0xab, 0x46, 0x9d, 
		0x8f, 0x4e, 0x87, 0xc5, 0xa9, 0xd8, 0xe2, 0xbf,
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, uuid); err != nil {
		conn.Close()
		return nil, err
	}

	log.Printf("VLESS handshake sent")
	return conn, nil
}

func forwardTUNtoVLESS(ctx context.Context, tunIface *tun.Interface, conn *websocket.Conn) {
	buf := make([]byte, bufferSize)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := tunIface.Read(buf)
			if err != nil {
				return
			}

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

			if _, err := tunIface.Write(packet); err != nil {
				return
			}
		}
	}
}