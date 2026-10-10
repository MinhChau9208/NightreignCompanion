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

// HTTPS to a game server sits idle between requests; that is not lag, so
// TCP flows get no gap figure (seen live: 4 s gaps on Nightreign's :443).
func TestNoGapForTCP(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	srv := netip.AddrPortFrom(peer, 443)
	local := netip.AddrPortFrom(me, 50000)
	base := int64(1_000_000_000)
	tr.Add(Packet{Proto: "tcp", Size: 900, TS: base, Src: srv, Dst: local})
	tr.Add(Packet{Proto: "tcp", Size: 900, TS: base + 4*ticksPerSecond, Src: srv, Dst: local})
	tr.Add(Packet{Proto: "tcp", Out: true, Size: 300, TS: base + 5*ticksPerSecond, Src: local, Dst: srv})
	f := tr.Snapshot(base+6*ticksPerSecond, 8)
	if len(f) != 1 || f[0].Kind != KindServer || f[0].MaxGapMs != 0 || f[0].PktsInPerSec == 0 {
		t.Errorf("tcp flow = %+v", f)
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
	cases := []struct{ proto, ip, want string }{
		{"udp", "162.254.193.6", KindRelay},
		{"udp", "203.0.113.7", KindPeer},
		{"tcp", "203.0.113.7", KindServer},
		{"udp", "10.0.0.5", KindLAN},
		{"tcp", "103.28.54.100", KindServer}, // Valve range, but TCP: Steam services, not a relay
	}
	for _, c := range cases {
		if got := classify(c.proto, netip.MustParseAddr(c.ip)); got != c.want {
			t.Errorf("classify(%s %s) = %s, want %s", c.proto, c.ip, got, c.want)
		}
	}
}

// mDNS/SSDP showed up as "peers" in a live capture; they are LAN discovery.
func TestMulticastSkipped(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	for _, dst := range []string{"224.0.0.251:5353", "239.255.255.250:1900", "[ff02::fb]:5353", "255.255.255.255:67"} {
		tr.Add(Packet{Proto: "udp", Out: true, TS: 1,
			Src: netip.AddrPortFrom(me, 5353), Dst: netip.MustParseAddrPort(dst)})
	}
	if n := len(tr.flows); n != 0 {
		t.Errorf("tracked %d multicast/broadcast flows", n)
	}
}

func TestViaAndSustained(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	base := int64(1_000_000_000)
	// Co-op through a relay: 20 packets/s each way for 5 s, sent by Steam.
	for i := int64(0); i < 100; i++ {
		ts := base + i*50*msTicks
		tr.Add(Packet{Proto: "udp", Out: true, TS: ts, Size: 190, Via: "steam.exe",
			Src: netip.AddrPortFrom(me, 1), Dst: netip.AddrPortFrom(relay, 4380)})
		tr.Add(Packet{Proto: "udp", TS: ts, Size: 170, Via: "steam.exe",
			Src: netip.AddrPortFrom(relay, 4380), Dst: netip.AddrPortFrom(me, 1)})
	}
	// A relay ping burst: a couple of packets, then nothing.
	other := netip.MustParseAddr("162.254.195.71")
	tr.Add(Packet{Proto: "udp", Out: true, TS: base, Via: "steam.exe",
		Src: netip.AddrPortFrom(me, 1), Dst: netip.AddrPortFrom(other, 27019)})
	tr.Add(Packet{Proto: "udp", TS: base + 80*msTicks, Via: "steam.exe",
		Src: netip.AddrPortFrom(other, 27019), Dst: netip.AddrPortFrom(me, 1)})

	flows := tr.Snapshot(base+5*ticksPerSecond, 8)
	if len(flows) != 2 {
		t.Fatalf("flows = %+v", flows)
	}
	if f := flows[0]; f.Via != "steam.exe" || f.Kind != KindRelay || !Sustained(f) {
		t.Errorf("session flow = %+v (sustained %v)", f, Sustained(f))
	}
	if f := flows[1]; Sustained(f) {
		t.Errorf("ping burst counted as sustained: %+v", f)
	}
}

// A co-op session through a relay whose traffic dips (a loading screen)
// must stay listed while it is still active, and Steam's ping bursts must
// never be listed.
func TestSessionFilterKeepsDippingSession(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	base := int64(1_000_000_000)
	send := func(ts int64, to netip.Addr, port uint16) {
		tr.Add(Packet{Proto: "udp", Out: true, TS: ts, Size: 190, Via: "steam.exe",
			Src: netip.AddrPortFrom(me, 1), Dst: netip.AddrPortFrom(to, port)})
		tr.Add(Packet{Proto: "udp", TS: ts + 30*msTicks, Size: 170, Via: "steam.exe",
			Src: netip.AddrPortFrom(to, port), Dst: netip.AddrPortFrom(me, 1)})
	}
	// 10 s at 20 packets/s each way, then 1 packet/s for 10 s, then silence.
	for i := int64(0); i < 200; i++ {
		send(base+i*50*msTicks, relay, 4380)
	}
	for i := int64(0); i < 10; i++ {
		send(base+10*ticksPerSecond+i*ticksPerSecond, relay, 4380)
	}
	other := netip.MustParseAddr("162.254.195.71")
	send(base+12*ticksPerSecond, other, 27019) // a relay ping burst

	var sf SessionFilter
	listed := func(sec int64) bool {
		flows := sf.Filter(tr.Snapshot(base+sec*ticksPerSecond, 16))
		for _, f := range flows {
			if f.IP == other.String() {
				t.Fatalf("%ds: ping burst listed: %+v", sec, f)
			}
		}
		return len(flows) == 1
	}
	for sec := int64(5); sec <= 20; sec++ {
		if !listed(sec) {
			t.Fatalf("%ds: session dropped while still active", sec)
		}
	}
	// 6 s after the last packet the session is idle and goes away.
	if listed(25) {
		t.Error("idle session still listed")
	}
	// An unfiltered rate check would have dropped it during the dip.
	if f := tr.Snapshot(base+18*ticksPerSecond, 16); Sustained(f[0]) {
		t.Fatalf("test does not exercise the dip: %+v", f[0])
	}
}

// The pattern measured in co-op: Steam sends the session through one relay
// and receives it through another, each with only acks the other way.
func TestSustainedAsymmetricRelays(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	base := int64(1_000_000_000)
	sendRelay := netip.AddrPortFrom(netip.MustParseAddr("103.28.54.162"), 4379)
	recvRelay := netip.AddrPortFrom(netip.MustParseAddr("103.28.54.178"), 4380)
	add := func(ts int64, out bool, r netip.AddrPort) {
		p := Packet{Proto: "udp", Out: out, TS: ts, Size: 180, Via: "steam.exe",
			Src: netip.AddrPortFrom(me, 1), Dst: r}
		if !out {
			p.Src, p.Dst = r, netip.AddrPortFrom(me, 1)
		}
		tr.Add(p)
	}
	for ms := int64(0); ms < 10_000; ms += 50 {
		ts := base + ms*msTicks
		add(ts, true, sendRelay)  // 20 pkt/s out
		add(ts, false, recvRelay) // 20 pkt/s in
		if ms%500 == 0 {
			add(ts, false, sendRelay) // acks: 2 pkt/s
			add(ts, true, recvRelay)
		}
	}
	// Acks a little under 2 pkt/s, as measured (1.8–1.9).
	flows := tr.Snapshot(base+11*ticksPerSecond, 8)
	if len(flows) != 2 {
		t.Fatalf("flows = %+v", flows)
	}
	for _, f := range flows {
		if min(f.PktsInPerSec, f.PktsOutPerSec) >= 2 {
			t.Fatalf("test does not exercise the ack side under 2 pkt/s: %+v", f)
		}
		if !Sustained(f) {
			t.Errorf("one-way relay route not sustained: %+v", f)
		}
		send := f.Remote == sendRelay.String()
		if f.SendOnly != send {
			t.Errorf("%s: sendOnly = %v, want %v", f.Remote, f.SendOnly, send)
		}
		// The acks' 500 ms spacing must not read as a lag spike.
		if send && f.MaxGapMs != 0 {
			t.Errorf("send route has a gap: %+v", f)
		}
		if !send && (f.MaxGapMs < 40 || f.MaxGapMs > 100) {
			t.Errorf("receive route gap = %v, want ~50 ms", f.MaxGapMs)
		}
	}
}

// A two-way route whose inbound side stalls is lag, not a send-only route.
func TestStallIsNotSendOnly(t *testing.T) {
	tr := NewTracker(10 * time.Second)
	tr.SetLocal([]netip.Addr{me})
	base := int64(1_000_000_000)
	r := netip.AddrPortFrom(relay, 4380)
	for ms := int64(0); ms < 20_000; ms += 50 {
		ts := base + ms*msTicks
		tr.Add(Packet{Proto: "udp", Out: true, TS: ts, Src: netip.AddrPortFrom(me, 1), Dst: r})
		if ms < 10_000 { // nothing comes back after 10 s
			tr.Add(Packet{Proto: "udp", TS: ts, Src: r, Dst: netip.AddrPortFrom(me, 1)})
		}
		if ms%1000 == 0 {
			tr.Snapshot(ts, 8) // the helper looks once per second
		}
	}
	f := tr.Snapshot(base+19_500*msTicks, 8)[0]
	if f.SendOnly || f.MaxGapMs < 9000 {
		t.Errorf("stall hidden: %+v", f)
	}
}
