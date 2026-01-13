package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

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
	resolver, _ := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		resolver.Browse(ctx, "_multiclip._tcp", "local.", entries)
	}()

	go func() {
		for entry := range entries {
			for _, ip := range entry.AddrIPv4 {
				onFound(fmt.Sprintf("%s:%d", ip, entry.Port))
			}
		}
	}()

	localIPs := getLocalIPs()
	for _, ip := range localIPs {
		go scanSubnet(ip, onFound)
	}
}

func scanSubnet(myIP string, onFound func(string)) {
	mask := net.ParseIP(myIP).To4()
	base := fmt.Sprintf("%d.%d.%d.", mask[0], mask[1], mask[2])

	for i := 1; i < 255; i++ {
		target := fmt.Sprintf("%s%d", base, i)
		if target == myIP {
			continue
		}
		go func(addr string) {
			conn, err := net.DialTimeout("tcp", addr+":8080", 500*time.Millisecond)
			if err == nil {
				conn.Close()
				onFound(addr + ":8080")
			}
		}(target)
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
