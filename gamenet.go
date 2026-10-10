package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/netip"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamenet"
	"github.com/MinhChau9208/NightreignCompanion/internal/netmon"
)

// Peers the main process pings at once (1 probe/s each), and how recently
// a flow must have carried traffic to be worth pinging.
const (
	maxPeerPings = 4
	peerActiveMs = 5000
)

// gameNetWatcher adds ping measurements to the helper's connection reports.
// Pinging needs no Administrator rights, so it runs here, not in the helper.
type gameNetWatcher struct {
	mu     sync.Mutex
	mon    *netmon.Monitor
	cancel context.CancelFunc
	key    string
	last   gamenet.Status
}

func (a *App) onGameNet(data json.RawMessage) {
	var st gamenet.Status
	if err := json.Unmarshal(data, &st); err != nil {
		log.Printf("ipc: bad game net stats: %v", err)
		return
	}
	w := &a.gnet
	w.mu.Lock()
	mon := w.monitor(a.ctx, a.cfg.Get().Network.Thresholds)
	mon.SetTargets(peerTargets(st.Flows))
	pings := map[string]netmon.Stats{}
	for _, s := range mon.Snapshot() {
		pings[s.Target] = s
	}
	attachPings(st.Flows, pings)
	w.last = st
	w.mu.Unlock()

	runtime.EventsEmit(a.ctx, gamenet.MsgStats, st)
	a.hub.Publish(gamenet.MsgStats, st)
}

// monitor returns the peer ping monitor, restarting it when the thresholds
// changed. Callers hold w.mu.
func (w *gameNetWatcher) monitor(parent context.Context, th config.Thresholds) *netmon.Monitor {
	key := fmt.Sprint(th)
	if w.mon != nil && w.key == key {
		return w.mon
	}
	if w.cancel != nil {
		w.cancel()
	}
	w.mon = netmon.NewMonitor(netmon.Config{
		Thresholds: netmon.Thresholds{PingMs: th.PingMs, JitterMs: th.JitterMs, LossPct: th.LossPct},
	})
	ctx, cancel := context.WithCancel(parent)
	w.cancel, w.key = cancel, key
	go w.mon.Run(ctx, nil)
	return w.mon
}

// peerTargets picks the busiest active UDP endpoints with an IPv4 address
// (the ICMP prober is IPv4-only). Relays also get their site router probed,
// as a stand-in when the relay itself ignores ICMP.
func peerTargets(flows []gamenet.Flow) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range flows {
		if len(seen) == maxPeerPings {
			break
		}
		if f.Proto != "udp" || f.IdleMs > peerActiveMs || seen[f.IP] {
			continue
		}
		if ip, err := netip.ParseAddr(f.IP); err != nil || !ip.Is4() {
			continue
		}
		seen[f.IP] = true
		out = append(out, f.IP)
		if r := siteRouter(f); r != "" {
			out = append(out, r)
		}
	}
	return out
}

// siteRouter is the .1 address of a relay's /24. Steam relays drop ICMP,
// but the router in front of them answers (measured: 103.28.54.185 silent,
// 103.28.54.1 at 74 ms), which approximates the latency to the relay.
func siteRouter(f gamenet.Flow) string {
	ip, err := netip.ParseAddr(f.IP)
	if f.Kind != gamenet.KindRelay || err != nil || !ip.Is4() {
		return ""
	}
	b := ip.As4()
	if b[3] == 1 {
		return ""
	}
	b[3] = 1
	return netip.AddrFrom4(b).String()
}

// attachPings puts each UDP flow's ping results on it, falling back to the
// relay's site router when the relay never answers.
func attachPings(flows []gamenet.Flow, pings map[string]netmon.Stats) {
	for i := range flows {
		f := &flows[i]
		p, ok := pings[f.IP]
		if f.Proto != "udp" || !ok {
			continue
		}
		f.Ping = &p
		if p.Received > 0 {
			continue
		}
		if r, ok := pings[siteRouter(*f)]; ok && r.Received > 0 {
			f.Ping, f.PingAddr = &r, r.Target
		}
	}
}

// clearGameNet stops pinging once the helper is gone.
func (a *App) clearGameNet() {
	w := &a.gnet
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
	}
	w.mon, w.cancel, w.key = nil, nil, ""
	w.last = gamenet.Status{}
}

// GameNet returns the latest game connection report (for the first paint
// before the next "game:net" event).
func (a *App) GameNet() gamenet.Status {
	a.gnet.mu.Lock()
	defer a.gnet.mu.Unlock()
	return a.gnet.last
}
