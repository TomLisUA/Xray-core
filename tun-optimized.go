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
	"github.com/goxray/core/network/route"
	"github.com/goxray/core/network/tun"
)

//go:embed tun.ko
var tunModule []byte

const (
	tunAddr = "10.50.0.2/24"
	// Optimized for E3372H - small buffers
	bufferSize = 2048
	maxConns   = 8
)

func main() {
	// E3372H optimizations
	optimizeForE3372H()
	
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <vps_domain:443>\n", os.Args[0])
		os.Exit(1)
	}

	vpsTarget := os.Args[1]
	
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunIface, err := setupTUN()
	if err != nil {
		log.Fatal("TUN setup failed:", err)
	}
	defer tunIface.Close()

	if err := setupRouting(vpsTarget); err != nil {
		log.Fatal("Routing setup failed:", err)
	}

	go startL3Tunnel(ctx, tunIface, vpsTarget)

	log.Printf("E3372H tunnel active: %s ⇄ %s", tunAddr, vpsTarget)
	<-sigterm
	cancel()
}

func optimizeForE3372H() {
	// Set OOM killer protection
	if f, err := os.OpenFile("/proc/self/oom_score_adj", os.O_WRONLY, 0); err == nil {
		f.WriteString("-1000")
		f.Close()
	}

	// Set high priority
	syscall.Setpriority(syscall.PRIO_PROCESS, 0, -20)

	// Limit Go runtime for low memory
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

	cmd := exec.Command("insmod", tmpFile)
	if err := cmd.Run(); err != nil {
		exec.Command("modprobe", "tun").Run()
	}

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

func setupRouting(vpsHost string) error {
	r, err := route.New()
	if err != nil {
		return err
	}

	vpsIP, err := net.ResolveIPAddr("ip", vpsHost[:len(vpsHost)-4])
	if err != nil {
		return err
	}

	gw := net.IPv4(192, 168, 1, 1)

	err = r.Add(route.Opts{
		Gateway: gw,
		Routes:  []*route.Addr{route.MustParseAddr(vpsIP.String() + "/32")},
	})
	if err != nil {
		return err
	}

	return r.Add(route.Opts{
		IfName: "tun-e3372h",
		Routes: []*route.Addr{
			route.MustParseAddr("0.0.0.0/1"),
			route.MustParseAddr("128.0.0.0/1"),
		},
	})
}

func startL3Tunnel(ctx context.Context, tunIface *tun.Interface, vpsTarget string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := dialVLESSWS(vpsTarget)
			if err != nil {
				time.Sleep(5 * time.Second)
				continue
			}

			go forwardTUNtoVLESS(ctx, tunIface, conn)
			forwardVLESStoTUN(ctx, conn, tunIface)
			
			conn.Close()
			time.Sleep(1 * time.Second)
		}
	}
}

func dialVLESSWS(target string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			ServerName: target[:len(target)-4],
		},
		HandshakeTimeout: 10 * time.Second,
	}

	headers := http.Header{}
	headers.Set("User-Agent", "Mozilla/5.0 (Linux; Android)")
	
	wsURL := fmt.Sprintf("wss://%s/vless", target)
	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return nil, err
	}

	vlessHandshake := []byte{
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // UUID
		0x00, // Version
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // Encryption
		0x00, // Reserved
		0x03, 0x00, // Command: TUN mode
		0x00, 0x00, // Port
		0x01, // Address type: IPv4
	}

	if err := conn.WriteMessage(websocket.BinaryMessage, vlessHandshake); err != nil {
		conn.Close()
		return nil, err
	}

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