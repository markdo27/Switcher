package localpeer

import (
	"net"
	"os"
	"testing"
	"time"
)

func TestWindowsSocketOwnership(t *testing.T) {
	for _, network := range []string{"tcp4", "tcp6"} {
		t.Run(network, func(t *testing.T) {
			addr := "127.0.0.1:0"
			if network == "tcp6" {
				addr = "[::1]:0"
			}
			ln, e := net.Listen(network, addr)
			if e != nil {
				t.Skip(e)
			}
			defer ln.Close()
			if uid, ok := ListenerUID(ln.Addr().(*net.TCPAddr)); !ok || uid != uint32(os.Getuid()) {
				t.Fatalf("listener uid=%d known=%v", uid, ok)
			}
			client, e := net.Dial(network, ln.Addr().String())
			if e != nil {
				t.Fatal(e)
			}
			defer client.Close()
			server, e := ln.Accept()
			if e != nil {
				t.Fatal(e)
			}
			defer server.Close()
			if uid, kind := peerSince(server, time.Now()); kind != peerFound || uid != uint32(os.Getuid()) {
				t.Fatalf("peer uid=%d kind=%v", uid, kind)
			}
			if e = ownConnection(client); e != nil {
				t.Fatal(e)
			}
		})
	}
}

func TestWindowsCrossProcessOwnership(t *testing.T) {
	for _, ipv6 := range []bool{false, true} {
		if got := selfTest(ipv6); got != working {
			t.Fatalf("cross-process socket lookup ipv6=%v: got %v want working", ipv6, got)
		}
	}
}
