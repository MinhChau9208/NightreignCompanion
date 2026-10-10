package gamenet

import (
	"encoding/binary"
	"math"
	"net/netip"
	"testing"
	"time"
)

const msTicks = ticksPerSecond / 1000

// payload builds a Kernel-Network send/receive payload as the provider lays it out.
func payload(pid, size uint32, daddr, saddr netip.Addr, dport, sport uint16) []byte {
	b := binary.LittleEndian.AppendUint32(nil, pid)
	b = binary.LittleEndian.AppendUint32(b, size)
	b = append(b, daddr.AsSlice()...)
	b = append(b, saddr.AsSlice()...)
	b = binary.BigEndian.AppendUint16(b, dport)
	b = binary.BigEndian.AppendUint16(b, sport)
	return append(b, make([]byte, 8)...) // seqnum, connid
}

var (
	me    = netip.MustParseAddr("192.168.1.10")
	relay = netip.MustParseAddr("155.133.230.34")
	peer  = netip.MustParseAddr("203.0.113.7")
)

func TestParse(t *testing.T) {
	p, ok := parse(42, payload(1234, 120, relay, me, 27015, 50000), 99)
	if !ok {
		t.Fatal("UDPv4 send not parsed")
	}
	want := Packet{PID: 1234, Proto: "udp", Out: true, Size: 120,
		Dst: netip.AddrPortFrom(relay, 27015), Src: netip.AddrPortFrom(me, 50000), TS: 99}
	if p != want {
		t.Errorf("got %+v\nwant %+v", p, want)
	}

	v6 := netip.MustParseAddr("2001:db8::1")
	p, ok = parse(27, payload(7, 60, netip.MustParseAddr("fe80::1"), v6, 443, 51000), 0)
	if !ok || p.Proto != "tcp" || p.Out || p.Src.Addr() != v6 || p.Src.Port() != 51000 {
		t.Errorf("TCPv6 receive = %+v, %v", p, ok)
	}

	if _, ok := parse(12, payload(1, 0, relay, me, 1, 2), 0); ok {
		t.Error("connect event should be ignored")
	}
	if _, ok := parse(42, []byte{1, 2, 3}, 0); ok {
		t.Error("short payload should be rejected")
	}
}

func TestRemoteSide(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	out := Packet{Proto: "udp", Out: true, Src: netip.AddrPortFrom(me, 1), Dst: netip.AddrPortFrom(peer, 2)}
	// A receive whose fields are named the "wrong" way round must still
	// resolve to the peer, since we decide by which side is local.
	inSwapped := Packet{Proto: "udp", Src: netip.AddrPortFrom(me, 1), Dst: netip.AddrPortFrom(peer, 2)}
	inManifest := Packet{Proto: "udp", Src: netip.AddrPortFrom(peer, 2), Dst: netip.AddrPortFrom(me, 1)}
	for _, p := range []Packet{out, inSwapped, inManifest} {
		r, ok := tr.remote(p)
		if !ok || r.Addr() != peer {
			t.Errorf("remote(%+v) = %v, %v", p, r, ok)
		}
	}
	loop := netip.MustParseAddr("127.0.0.1")
	if _, ok := tr.remote(Packet{Src: netip.AddrPortFrom(loop, 1), Dst: netip.AddrPortFrom(loop, 2)}); ok {
		t.Error("loopback traffic should be skipped")
	}
}

func approx(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestSnapshot(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	base := int64(1_000_000_000)
	add := func(remote netip.Addr, out bool, ts int64) {
		p := Packet{Proto: "udp", Out: out, Size: 125, TS: ts}
		if out {
			p.Src, p.Dst = netip.AddrPortFrom(me, 5000), netip.AddrPortFrom(remote, 27015)
		} else {
			p.Src, p.Dst = netip.AddrPortFrom(remote, 27015), netip.AddrPortFrom(me, 5000)
		}
		tr.Add(p)
	}

	// Relay: 10 s of traffic at 20 packets/s each way, with one 400 ms hole
	// in what comes back. Delivered out of order to exercise insertion.
	for i := int64(199); i >= 0; i-- {
		ts := base + i*50*msTicks
		add(relay, true, ts)
		if i < 100 || i >= 108 {
			add(relay, false, ts)
		}
	}
	// Peer: a quiet flow that stopped answering 1.5 s before our last send.
	add(peer, false, base+8*ticksPerSecond)
	add(peer, true, base+8*ticksPerSecond)
	add(peer, true, base+9500*msTicks)

	now := base + 10*ticksPerSecond
	flows := tr.Snapshot(now, 8)
	if len(flows) != 2 {
		t.Fatalf("got %d flows: %+v", len(flows), flows)
	}
	r := flows[0]
	if r.IP != relay.String() || r.Kind != KindRelay || r.Proto != "udp" {
		t.Fatalf("busiest flow = %+v", r)
	}
	if !approx(r.PktsOutPerSec, 20) || !approx(r.PktsInPerSec, 19.2) {
		t.Errorf("rates in/out = %.2f/%.2f", r.PktsInPerSec, r.PktsOutPerSec)
	}
	if !approx(r.KbpsOut, 20) { // 20 pkt/s × 125 B × 8 bit
		t.Errorf("kbps out = %.2f", r.KbpsOut)
	}
	if !approx(r.MaxGapMs, 450) { // 9 intervals of 50 ms across the hole
		t.Errorf("max gap = %.1f ms", r.MaxGapMs)
	}

	p := flows[1]
	if p.Kind != KindPeer || !approx(p.MaxGapMs, 1500) {
		t.Errorf("peer flow = %+v", p)
	}
	if !approx(p.IdleMs, 500) {
		t.Errorf("peer idle = %.1f ms", p.IdleMs)
	}

	// Everything ages out of the window.
	if flows := tr.Snapshot(now+11*ticksPerSecond, 8); len(flows) != 0 {
		t.Errorf("stale flows kept: %+v", flows)
	}
}

func TestFlowLimit(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	for i := range maxFlows + 10 {
		tr.Add(Packet{Proto: "udp", Out: true, TS: 1,
			Src: netip.AddrPortFrom(me, 1), Dst: netip.AddrPortFrom(peer, uint16(1000+i))})
	}
	if n := len(tr.flows); n != maxFlows {
		t.Errorf("tracked %d flows, want cap %d", n, maxFlows)
	}
	if got := tr.Snapshot(2, 5); len(got) != 5 {
		t.Errorf("limit not applied: %d flows", len(got))
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]struct{ proto, ip string }{
		KindRelay:  {"udp", "162.254.193.6"},
		KindPeer:   {"udp", "203.0.113.7"},
		KindServer: {"tcp", "203.0.113.7"},
		KindLAN:    {"udp", "10.0.0.5"},
	}
	for want, c := range cases {
		if got := classify(c.proto, netip.MustParseAddr(c.ip)); got != want {
			t.Errorf("classify(%s %s) = %s, want %s", c.proto, c.ip, got, want)
		}
	}
}
