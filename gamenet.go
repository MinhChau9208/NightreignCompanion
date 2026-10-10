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
	for i := range st.Flows {
		if p, ok := pings[st.Flows[i].IP]; ok && st.Flows[i].Proto == "udp" {
			st.Flows[i].Ping = &p
		}
	}
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
// (the ICMP prober is IPv4-only).
func peerTargets(flows []gamenet.Flow) []string {
	var out []string
	seen := map[string]bool{}
	for _, f := range flows {
		if len(out) == maxPeerPings {
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
	}
	return out
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
