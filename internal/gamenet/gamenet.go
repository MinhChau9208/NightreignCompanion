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
	// A spike here is what a lag spike looks like on the wire. 0 = no data.
	MaxGapMs float64 `json:"maxGapMs"`
	IdleMs   float64 `json:"idleMs"` // since the last packet in either direction
	AgeSec   float64 `json:"ageSec"` // since the first packet seen

	// Ping is filled in by the main process, which probes active peers.
	Ping *netmon.Stats `json:"ping,omitempty"`
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
	in, out     []sample // sorted by ts
	first, last int64
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
	switch {
	case sl && dl:
		return netip.AddrPort{}, false // loopback or to ourselves
	case sl:
		return p.Dst, true
	case dl:
		return p.Src, true
	case p.Out:
		return p.Dst, true
	default:
		return p.Src, true
	}
}

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
		f = &flow{first: p.TS, last: p.TS}
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

	if len(f.in) > 0 {
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

func ticksToMs(t int64) float64 { return float64(t) / float64(ticksPerSecond/1000) }
