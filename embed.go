package main

import (
	_ "embed"
	"os"
	"os/exec"
	"syscall"
)

//go:embed tun.ko
var tunModule []byte

func setupTUNModule() error {
	// Create /dev/net directory
	if err := os.MkdirAll("/dev/net", 0755); err != nil {
		return err
	}

	// Create TUN device node
	if err := syscall.Mknod("/dev/net/tun", syscall.S_IFCHR|0666, int(syscall.Mkdev(10, 200))); err != nil {
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