// Package gamenet watches the game's own network traffic (M1, docs/SCOPE.md
// §3): which remote endpoints it talks to, how much, and how steadily.
//
// Packets come from the Microsoft-Windows-Kernel-Network ETW provider,
// which reports every TCP/UDP send and receive with the owning PID. That is
// the OS network stack's view, so nothing touches the game. UDP is
// connectionless, so the OS connection tables (GetExtendedUdpTable) only
// know local ports; ETW is the only safe source of the peer/relay address.
package gamenet

import (
	"cmp"
	"encoding/binary"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/netmon"
)

// MsgStats is the helper → main message carrying a Status once per second.
const MsgStats = "game:net"

// Status is what the helper reports about the game's connections.
type Status struct {
	State string `json:"state"` // same states as the FPS report: starting | waiting | running | error
	PID   uint32 `json:"pid"`
	Flows []Flow `json:"flows"`
	Error string `json:"error,omitempty"`
}

// Flow kinds.
const (
	KindRelay  = "relay"  // Steam Datagram Relay (Valve address space)
	KindPeer   = "peer"   // UDP to a public address: another player, or a non-Valve relay
	KindServer = "server" // TCP: matchmaking / online services
	KindLAN    = "lan"    // private or link-local address
)

// Flow summarises traffic to one remote endpoint over the window.
type Flow struct {
	Proto  string `json:"proto"` // udp | tcp
	Remote string `json:"remote"`
	IP     string `json:"ip"`
	Kind   string `json:"kind"`

	PktsInPerSec  float64 `json:"pktsInPerSec"`
	PktsOutPerSec float64 `json:"pktsOutPerSec"`
	KbpsIn        float64 `json:"kbpsIn"`
	KbpsOut       float64 `json:"kbpsOut"`
	// MaxGapMs is the longest silence from the remote side in the window,
	// including a trailing gap while we keep sending but nothing comes back.
	// A spike here is what a lag spike looks like on the wire. UDP only:
	// request/response TCP is silent between requests by design (measured
	// on Nightreign's HTTPS connections: 4 s gaps while idle). 0 = no data.
	MaxGapMs float64 `json:"maxGapMs"`
	// SendOnly marks a route Steam uses only to send the session: it comes
	// back through another relay, so all this one returns is acks a few
	// times per second, and its silence gaps say nothing about lag.
	// MaxGapMs is 0 on such a flow.
	SendOnly bool    `json:"sendOnly,omitempty"`
	IdleMs   float64 `json:"idleMs"` // since the last packet in either direction
	AgeSec   float64 `json:"ageSec"` // since the first packet seen

	// Via names the process that carries the flow when it is not the game
	// itself: Nightreign's co-op traffic is sent by steam.exe (Steam
	// networking through a Valve relay), not by nightreign.exe.
	Via string `json:"via,omitempty"`

	// Ping is filled in by the main process, which probes active peers.
	// PingAddr is set when the probe went to a stand-in address because the
	// remote itself ignores ICMP (Steam relays do).
	Ping     *netmon.Stats `json:"ping,omitempty"`
	PingAddr string        `json:"pingAddr,omitempty"`
}

// Packet is one send or receive as reported by the OS.
type Packet struct {
	PID   uint32
	Proto string
	Out   bool // sent by this machine
	Size  uint32
	Src   netip.AddrPort // saddr/sport of the event
	Dst   netip.AddrPort // daddr/dport of the event
	TS    int64          // FILETIME ticks (100 ns)
	Via   string         // set by the caller, see Flow.Via
}

// Kernel-Network event IDs (from the provider manifest).
var eventKinds = map[uint16]struct {
	proto string
	out   bool
	v6    bool
}{
	10: {"tcp", true, false}, 11: {"tcp", false, false},
	26: {"tcp", true, true}, 27: {"tcp", false, true},
	42: {"udp", true, false}, 43: {"udp", false, false},
	58: {"udp", true, true}, 59: {"udp", false, true},
}

// parse decodes a Kernel-Network send/receive payload. Every template
// starts with PID, size, daddr, saddr, dport, sport; addresses are 4 or 16
// bytes and ports are in network byte order.
func parse(id uint16, data []byte, ts int64) (Packet, bool) {
	k, ok := eventKinds[id]
	if !ok {
		return Packet{}, false
	}
	alen := 4
	if k.v6 {
		alen = 16
	}
	if len(data) < 8+2*alen+4 {
		return Packet{}, false
	}
	p := Packet{
		PID:   binary.LittleEndian.Uint32(data[0:]),
		Size:  binary.LittleEndian.Uint32(data[4:]),
		Proto: k.proto,
		Out:   k.out,
		TS:    ts,
	}
	dst, _ := netip.AddrFromSlice(data[8 : 8+alen])
	src, _ := netip.AddrFromSlice(data[8+alen : 8+2*alen])
	ports := data[8+2*alen:]
	p.Dst = netip.AddrPortFrom(dst.Unmap(), binary.BigEndian.Uint16(ports[0:]))
	p.Src = netip.AddrPortFrom(src.Unmap(), binary.BigEndian.Uint16(ports[2:]))
	return p, true
}

// Timestamps are FILETIME ticks (100 ns).
const ticksPerSecond = int64(time.Second / 100)

// maxFlows bounds memory if the game sprays many endpoints (e.g. a NAT
// punch-through burst); further new endpoints are ignored until old ones age out.
const maxFlows = 64

type sample struct {
	ts   int64
	size uint32
}

type flowKey struct {
	proto  string
	remote netip.AddrPort
}

type flow struct {
	via         string
	in, out     []sample // sorted by ts
	first, last int64
	// recvd is set once a stream has come in, so that a later stall on a
	// two-way route is reported as a gap rather than as a send-only route.
	recvd bool
}

// Tracker aggregates the packets of one process per remote endpoint.
type Tracker struct {
	mu     sync.Mutex
	window int64
	local  map[netip.Addr]bool
	flows  map[flowKey]*flow
}

func NewTracker(window time.Duration) *Tracker {
	return &Tracker{window: int64(window / 100), flows: map[flowKey]*flow{}}
}

// SetLocal sets this machine's addresses, used to tell which side of a
// packet is the remote one.
func (t *Tracker) SetLocal(addrs []netip.Addr) {
	m := make(map[netip.Addr]bool, len(addrs))
	for _, a := range addrs {
		m[a.Unmap()] = true
	}
	t.mu.Lock()
	t.local = m
	t.mu.Unlock()
}

func (t *Tracker) isLocal(a netip.Addr) bool {
	return a.IsLoopback() || a.IsUnspecified() || t.local[a]
}

// remote picks the far end of p. The manifest describes receives as "from
// saddr to daddr", but rather than trust field naming we check which side
// is one of our addresses, and fall back to the manifest only when neither is.
func (t *Tracker) remote(p Packet) (netip.AddrPort, bool) {
	sl, dl := t.isLocal(p.Src.Addr()), t.isLocal(p.Dst.Addr())
	var r netip.AddrPort
	switch {
	case sl && dl:
		return netip.AddrPort{}, false // loopback or to ourselves
	case sl:
		r = p.Dst
	case dl:
		r = p.Src
	case p.Out:
		r = p.Dst
	default:
		r = p.Src
	}
	// Multicast/broadcast (mDNS, SSDP, …) is LAN discovery, not a remote host.
	if a := r.Addr(); a.IsMulticast() || a == broadcast {
		return netip.AddrPort{}, false
	}
	return r, true
}

var broadcast = netip.AddrFrom4([4]byte{255, 255, 255, 255})

// Sustained reports whether a flow carries a steady game stream, as opposed
// to Steam's relay ping bursts (a few packets in all). Steam may route the
// two directions of a session through different relays, so only one side
// has to be a stream; the other may be just acknowledgements (measured in
// a co-op session: ~19 pkt/s out and ~1.9 back on one relay, ~16 in and
// ~1.9 out on another).
func Sustained(f Flow) bool {
	hi, lo := max(f.PktsInPerSec, f.PktsOutPerSec), min(f.PktsInPerSec, f.PktsOutPerSec)
	return f.AgeSec >= 3 && hi >= streamPkts && lo >= 0.5
}

// keepIdleMs is how long a flow that was once Sustained stays listed while
// it carries no traffic at all.
const keepIdleMs = 5000

// SessionFilter keeps the game's own flows and the ones another process
// (Steam) carries for it, dropping that process's other chatter. A carried
// flow must be Sustained to be listed, but once it has been, it stays while
// it is still active: Sustained is judged on rates over the window, and the
// session's rate dips (loading screens, quiet moments) must not make the
// row blink in and out. Not safe for concurrent use.
type SessionFilter struct {
	kept map[string]bool // proto + remote
}

// Filter returns the flows to report, reusing the backing array of flows.
func (s *SessionFilter) Filter(flows []Flow) []Flow {
	if s.kept == nil {
		s.kept = map[string]bool{}
	}
	present := make(map[string]bool, len(flows))
	out := flows[:0]
	for _, f := range flows {
		if f.Via == "" {
			out = append(out, f)
			continue
		}
		k := f.Proto + " " + f.Remote
		present[k] = true
		switch {
		case Sustained(f):
			s.kept[k] = true
		case s.kept[k] && f.IdleMs < keepIdleMs:
		default:
			delete(s.kept, k)
			continue
		}
		out = append(out, f)
	}
	for k := range s.kept {
		if !present[k] {
			delete(s.kept, k) // aged out of the tracker
		}
	}
	return out
}

// Reset forgets every kept flow (the game restarted).
func (s *SessionFilter) Reset() { clear(s.kept) }

// Add records one packet. ETW may deliver events slightly out of order
// across CPU buffers, so samples are inserted in place.
func (t *Tracker) Add(p Packet) {
	t.mu.Lock()
	defer t.mu.Unlock()
	r, ok := t.remote(p)
	if !ok {
		return
	}
	k := flowKey{p.Proto, r}
	f := t.flows[k]
	if f == nil {
		if len(t.flows) >= maxFlows {
			return
		}
		f = &flow{first: p.TS, last: p.TS, via: p.Via}
		t.flows[k] = f
	}
	f.first, f.last = min(f.first, p.TS), max(f.last, p.TS)
	s := sample{p.TS, p.Size}
	if p.Out {
		f.out = insert(f.out, s)
	} else {
		f.in = insert(f.in, s)
	}
}

func insert(s []sample, x sample) []sample {
	if n := len(s); n == 0 || x.ts >= s[n-1].ts {
		return append(s, x)
	}
	i, _ := slices.BinarySearchFunc(s, x.ts, func(e sample, ts int64) int { return cmp.Compare(e.ts, ts) })
	return slices.Insert(s, i, x)
}

func (t *Tracker) Reset() {
	t.mu.Lock()
	clear(t.flows)
	t.mu.Unlock()
}

// Snapshot drops samples older than the window and returns up to max
// flows, busiest first.
func (t *Tracker) Snapshot(now int64, limit int) []Flow {
	t.mu.Lock()
	defer t.mu.Unlock()
	cut := now - t.window
	out := make([]Flow, 0, len(t.flows))
	for k, f := range t.flows {
		f.in, f.out = trim(f.in, cut), trim(f.out, cut)
		if len(f.in) == 0 && len(f.out) == 0 {
			delete(t.flows, k)
			continue
		}
		out = append(out, summarize(k, f, now, t.window))
	}
	slices.SortFunc(out, func(a, b Flow) int {
		return cmp.Or(
			cmp.Compare(b.PktsInPerSec+b.PktsOutPerSec, a.PktsInPerSec+a.PktsOutPerSec),
			cmp.Compare(a.Remote, b.Remote),
		)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func trim(s []sample, cut int64) []sample {
	i, _ := slices.BinarySearchFunc(s, cut, func(e sample, ts int64) int { return cmp.Compare(e.ts, ts) })
	return slices.Delete(s, 0, i)
}

func summarize(k flowKey, f *flow, now, window int64) Flow {
	fl := Flow{
		Proto:  k.proto,
		Remote: k.remote.String(),
		IP:     k.remote.Addr().String(),
		Kind:   classify(k.proto, k.remote.Addr()),
		Via:    f.via,
		IdleMs: ticksToMs(max(0, now-f.last)),
		AgeSec: float64(max(0, now-f.first)) / float64(ticksPerSecond),
	}
	// Rates over the part of the window the flow has existed, at least 1 s
	// so a brand-new flow does not show a huge rate.
	span := float64(min(window, max(ticksPerSecond, now-f.first))) / float64(ticksPerSecond)
	rate := func(s []sample) (pkts, kbps float64) {
		var bytes uint64
		for _, x := range s {
			bytes += uint64(x.size)
		}
		return float64(len(s)) / span, float64(bytes) * 8 / 1000 / span
	}
	fl.PktsInPerSec, fl.KbpsIn = rate(f.in)
	fl.PktsOutPerSec, fl.KbpsOut = rate(f.out)

	f.recvd = f.recvd || fl.PktsInPerSec >= streamPkts
	fl.SendOnly = k.proto == "udp" && !f.recvd && sendOnly(fl.PktsInPerSec, fl.PktsOutPerSec)
	if k.proto == "udp" && len(f.in) > 0 && !fl.SendOnly {
		var gap int64
		for i := 1; i < len(f.in); i++ {
			gap = max(gap, f.in[i].ts-f.in[i-1].ts)
		}
		// Still sending but nothing back: that silence counts too. Measured
		// against our last send, not now, because ETW delivers in ~1 s batches.
		if n := len(f.out); n > 0 {
			gap = max(gap, f.out[n-1].ts-f.in[len(f.in)-1].ts)
		}
		fl.MaxGapMs = ticksToMs(gap)
	}
	return fl
}

// sendOnly reports a stream going out with only acks coming back
// (measured: ~19 pkt/s out, ~1.9 in). A receive route never qualifies, as
// its outbound side is the acks, so a stall on it still shows as a gap.
func sendOnly(in, out float64) bool { return out >= streamPkts && in < out/4 }

// streamPkts is the rate, in packets/s, from which one direction of a flow
// is a game stream rather than acks or pings.
const streamPkts = 5

func ticksToMs(t int64) float64 { return float64(t) / float64(ticksPerSecond/1000) }
