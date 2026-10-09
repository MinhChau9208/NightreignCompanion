package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/MinhChau9208/NightreignCompanion/internal/config"
	"github.com/MinhChau9208/NightreignCompanion/internal/netmon"
)

// netWatcher owns the running network monitor and restarts it when the
// targets or thresholds change.
type netWatcher struct {
	mu     sync.Mutex
	mon    *netmon.Monitor
	cancel context.CancelFunc
	key    string
}

// startNet (re)starts monitoring for s; a no-op if nothing relevant changed.
func (a *App) startNet(s config.NetworkSettings) {
	w := &a.net
	key := fmt.Sprint(s.PingTargets, s.Thresholds)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.mon != nil && w.key == key {
		return
	}
	if w.cancel != nil {
		w.cancel()
	}
	th := s.Thresholds
	mon := netmon.NewMonitor(netmon.Config{
		Targets:    s.PingTargets,
		Thresholds: netmon.Thresholds{PingMs: th.PingMs, JitterMs: th.JitterMs, LossPct: th.LossPct},
	})
	ctx, cancel := context.WithCancel(a.ctx)
	w.mon, w.cancel, w.key = mon, cancel, key

	go mon.Run(ctx, func(stats []netmon.Stats) {
		// A replaced monitor may tick once more while shutting down.
		if ctx.Err() != nil {
			return
		}
		runtime.EventsEmit(a.ctx, "net:stats", stats)
		a.hub.Publish("net:stats", stats)
	})
}

// NetStats returns the latest network stats (for the first paint before
// the next "net:stats" event).
func (a *App) NetStats() []netmon.Stats {
	a.net.mu.Lock()
	mon := a.net.mon
	a.net.mu.Unlock()
	if mon == nil {
		return []netmon.Stats{}
	}
	return mon.Snapshot()
}
