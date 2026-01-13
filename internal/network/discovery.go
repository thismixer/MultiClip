package network

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/grandcat/zeroconf"
)

func Advertise(ctx context.Context, port int) {
	server, err := zeroconf.Register("MultiClip-Device", "_multiclip._tcp", "local.", port, []string{"txtv=0", "lo=1"}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Shutdown()

	<-ctx.Done()
}

func Discover(ctx context.Context, onFound func(addr string)) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatal(err)
	}

	localIPs := getLocalIPs()
	fmt.Printf("Локальные IP: %v\n", localIPs)

	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		err = resolver.Browse(ctx, "_multiclip._tcp", "local.", entries)
		if err != nil {
			log.Fatal(err)
		}
	}()

	for entry := range entries {
		for _, ip := range entry.AddrIPv4 {
			ipStr := ip.String()
			if isLocal(ipStr, localIPs) {
				continue
			}
			addr := fmt.Sprintf("%s:%d", ipStr, entry.Port)
			onFound(addr)
		}
	}
}

func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}

func isLocal(ip string, localIPs []string) bool {
	for _, localIP := range localIPs {
		if ip == localIP {
			return true
		}
	}
	return false
}
