package network

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

func Advertise(ctx context.Context, port int) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	message := []byte(fmt.Sprintf("MCLIP_PEER:%d", port))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ifaces, _ := net.Interfaces()
			for _, iface := range ifaces {
				if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
					continue
				}

				addrs, _ := iface.Addrs()
				for _, addr := range addrs {
					ipnet, ok := addr.(*net.IPNet)
					if !ok || ipnet.IP.To4() == nil {
						continue
					}

					localAddr := &net.UDPAddr{IP: ipnet.IP, Port: 0}
					remoteAddr, _ := net.ResolveUDPAddr("udp", "255.255.255.255:9999")

					conn, err := net.DialUDP("udp", localAddr, remoteAddr)
					if err == nil {
						_, _ = conn.Write(message)
						conn.Close()
					}
				}
			}
		}
	}
}

func Discover(ctx context.Context, onPeerFound func(string)) {
	myIPs := make(map[string]bool)
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				myIPs[ipnet.IP.String()] = true
			}
		}
	}

	addr, _ := net.ResolveUDPAddr("udp", ":9999")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Printf("Ошибка прослушивания UDP: %v\n", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			if myIPs[remoteAddr.IP.String()] {
				continue
			}

			msg := string(buffer[:n])
			if strings.HasPrefix(msg, "MCLIP_PEER:") {
				peerPort := strings.TrimPrefix(msg, "MCLIP_PEER:")
				peerFullAddr := net.JoinHostPort(remoteAddr.IP.String(), peerPort)
				onPeerFound(peerFullAddr)
			}
		}
	}
}
