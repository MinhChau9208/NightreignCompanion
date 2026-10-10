package gamenet

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/etw"
)

// With elevation, a UDP datagram this test sends must come back through
// ETW as a parsed send event with our PID and the right destination.
// Without elevation it skips, like the other ETW tests.
func TestKernelNetworkLive(t *testing.T) {
	// TEST-NET-1: never routed, but the send is still logged.
	dst := netip.MustParseAddrPort("192.0.2.1:9")
	got := make(chan Packet, 16)
	pid := uint32(os.Getpid())
	sess, err := etw.StartSession("NightreignCompanion-NetTest", func(e *etw.Event) {
		if p, ok := ParseEvent(e); ok && p.PID == pid && p.Out && p.Proto == "udp" {
			select {
			case got <- p:
			default:
			}
		}
	})
	if errors.Is(err, etw.ErrNeedsAdmin) {
		t.Skip("not elevated: got ErrNeedsAdmin as expected")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	if err := Enable(sess); err != nil {
		t.Fatal(err)
	}
	go sess.Process()

	conn, err := net.DialUDP("udp4", nil, net.UDPAddrFromAddrPort(dst))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	deadline := time.After(10 * time.Second)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case p := <-got:
			if p.Dst != dst || p.Size < 5 {
				t.Fatalf("parsed %+v, want dst %v and at least 5 bytes", p, dst)
			}
			return
		case <-tick.C:
			conn.Write([]byte("hello"))
		case <-deadline:
			t.Fatal("no UDP send event within 10s")
		}
	}
}
