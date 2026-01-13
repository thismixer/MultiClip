package network

import (
	"context"
	"fmt"
	"log"

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

	entries := make(chan *zeroconf.ServiceEntry)
	go func() {
		err = resolver.Browse(ctx, "_multiclip._tcp", "local.", entries)
		if err != nil {
			log.Fatal(err)
		}
	}()

	for entry := range entries {
		if len(entry.AddrIPv4) > 0 {
			addr := fmt.Sprintf("%s:%d", entry.AddrIPv4[0], entry.Port)
			onFound(addr)
		}
	}
}
