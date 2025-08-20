package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goxray/core/network/tun"
)

const (
	tunAddr    = "10.50.0.2/24"
	modemAddr  = "192.168.24.1:80"
	proxyPort  = ":8080"
	vpsAddr    = "YOUR_VPS_IP:PORT" // Zastąp swoim adresem VPS
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <vps_address:port>\n", os.Args[0])
		os.Exit(1)
	}
	
	vpsTarget := os.Args[1]
	
	// Setup signal handling
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, os.Interrupt, syscall.SIGTERM)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	log.Println("Starting tun-simpl...")
	
	// Create TUN interface
	tunIface, err := setupTUN()
	if err != nil {
		log.Fatal("Failed to setup TUN:", err)
	}
	defer tunIface.Close()
	
	// Start services
	go startRawSocketListener(ctx, vpsTarget)
	go startHTTPProxy(ctx)
	go handleTUNTraffic(ctx, tunIface, vpsTarget)
	
	log.Println("tun-simpl started successfully")
	log.Printf("TUN interface: %s", tunAddr)
	log.Printf("HTTP proxy: http://10.50.0.2%s", proxyPort)
	log.Printf("VPS target: %s", vpsTarget)
	
	<-sigterm
	log.Println("Shutting down...")
	cancel()
	time.Sleep(100 * time.Millisecond)
}

// setupTUN creates and configures TUN interface
func setupTUN() (*tun.Interface, error) {
	iface, err := tun.New("tun-simpl", 1500)
	if err != nil {
		return nil, fmt.Errorf("create TUN: %w", err)
	}
	
	// Parse TUN address
	ip, ipnet, err := net.ParseCIDR(tunAddr)
	if err != nil {
		return nil, fmt.Errorf("parse TUN address: %w", err)
	}
	
	if err := iface.Up(&net.IPNet{IP: ip, Mask: ipnet.Mask}, ip); err != nil {
		return nil, fmt.Errorf("bring TUN up: %w", err)
	}
	
	log.Printf("TUN interface %s created with address %s", iface.Name(), tunAddr)
	return iface, nil
}

// startRawSocketListener listens for packets from VPS and forwards to WAN
func startRawSocketListener(ctx context.Context, vpsAddr string) {
	conn, err := net.ListenPacket("udp", "10.50.0.2:0")
	if err != nil {
		log.Printf("Failed to create raw socket listener: %v", err)
		return
	}
	defer conn.Close()
	
	log.Println("Raw socket listener started")
	
	for {
		select {
		case <-ctx.Done():
			return
		default:
			buf := make([]byte, 1500)
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Raw socket read error: %v", err)
				continue
			}
			
			log.Printf("Received %d bytes from %s", n, addr)
			// Forward to WAN interface (simplified)
			go sendToWAN(buf[:n])
		}
	}
}

// sendToWAN forwards packets to WAN interface
func sendToWAN(data []byte) {
	// Simplified WAN forwarding - in real implementation this would
	// interact with the actual network interface
	log.Printf("Forwarding %d bytes to WAN", len(data))
	// TODO: Implement actual WAN forwarding based on your network setup
}

// startHTTPProxy starts HTTP proxy for modem web interface
func startHTTPProxy(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(w, r, modemAddr)
	})
	
	server := &http.Server{
		Addr:    "10.50.0.2" + proxyPort,
		Handler: mux,
	}
	
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()
	
	log.Printf("HTTP proxy started on http://10.50.0.2%s", proxyPort)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("HTTP proxy error: %v", err)
	}
}

// proxyRequest proxies HTTP requests to target address
func proxyRequest(w http.ResponseWriter, r *http.Request, target string) {
	// Create target URL
	targetURL := &url.URL{
		Scheme: "http",
		Host:   target,
		Path:   r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}
	
	// Create new request
	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create proxy request", http.StatusInternalServerError)
		return
	}
	
	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}
	
	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to proxy request", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	
	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleTUNTraffic processes raw IP packets from TUN and forwards via VLESS+WS
func handleTUNTraffic(ctx context.Context, tunIface *tun.Interface, vpsAddr string) {
	log.Println("TUN L3 handler started")
	
	// WebSocket connection to VPS with VLESS protocol
	conn, err := dialVLESS(vpsAddr)
	if err != nil {
		log.Printf("Failed to connect VLESS to %s: %v", vpsAddr, err)
		return
	}
	defer conn.Close()
	
	log.Printf("VLESS+WS connected: %s", vpsAddr)
	
	// Forward raw IP packets TUN → VLESS
	go func() {
		buf := make([]byte, 1500)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := tunIface.Read(buf)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					continue
				}
				
				// Send raw IP packet through VLESS tunnel
				if _, err := conn.Write(buf[:n]); err != nil {
					log.Printf("VLESS write error: %v", err)
					return
				}
			}
		}
	}()
	
	// Forward VLESS → TUN (preserve original packets)
	buf := make([]byte, 1500)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, err := conn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if ctx.Err() != nil {
					return
				}
				return
			}
			
			// Write raw IP packet back to TUN
			if _, err := tunIface.Write(buf[:n]); err != nil {
				log.Printf("TUN write error: %v", err)
				return
			}
		}
	}
}

// dialVLESS creates VLESS+WebSocket connection (simplified)
func dialVLESS(addr string) (net.Conn, error) {
	// TODO: Implement proper VLESS+WS handshake
	// For now, use plain TCP (should be WebSocket with VLESS protocol)
	return net.Dial("tcp", addr)
}
}