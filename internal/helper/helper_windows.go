package helper

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"

	"github.com/MinhChau9208/NightreignCompanion/internal/etw"
	"github.com/MinhChau9208/NightreignCompanion/internal/fps"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamenet"
	"github.com/MinhChau9208/NightreignCompanion/internal/ipc"
)

// localRefresh is how often the machine's own addresses are re-read
// (Wi-Fi reconnects, VPNs coming up).
const localRefresh = 10 * time.Second

// Run measures the target process and reports to the main process until
// told to stop or the main process goes away.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Window <= 0 {
		cfg.Window = 30 * time.Second
	}
	if cfg.NetWindow <= 0 {
		cfg.NetWindow = 10 * time.Second
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	postFPS := func(st fps.Status) error { return ipc.Post(ctx, cfg.Addr, cfg.Token, fps.MsgStats, st) }
	postNet := func(st gamenet.Status) error { return ipc.Post(ctx, cfg.Addr, cfg.Token, gamenet.MsgStats, st) }
	if err := postFPS(fps.Status{State: fps.StateStarting, Process: cfg.Process}); err != nil {
		return err // main process unreachable; nothing to report to
	}

	// Leave when the main process says so or its event stream ends.
	go func() {
		ipc.Subscribe(ctx, cfg.Addr, cfg.Token, func(m ipc.Message) {
			if m.Type == MsgStop {
				cancel()
			}
		})
		cancel()
	}()

	var pid, steamPID atomic.Uint32
	frames := fps.NewTracker(cfg.Window)
	conns := gamenet.NewTracker(cfg.NetWindow)
	sess, err := etw.StartSession(SessionName, func(e *etw.Event) {
		if p, ts, ok := fps.PresentEvent(e); ok {
			if p != 0 && p == pid.Load() {
				frames.Add(ts)
			}
			return
		}
		pk, ok := gamenet.ParseEvent(e)
		if !ok || pk.PID == 0 {
			return
		}
		switch {
		case pk.PID == pid.Load():
			conns.Add(pk)
		case pk.PID == steamPID.Load() && pk.Proto == "udp":
			pk.Via = steamProcess
			conns.Add(pk)
		}
	})
	if err == nil {
		if err = fps.EnablePresents(sess); err != nil {
			sess.Close()
		}
	}
	if err != nil {
		postFPS(fps.Status{State: fps.StateError, Process: cfg.Process, Error: err.Error()})
		postNet(gamenet.Status{State: fps.StateError, Error: err.Error()})
		return err
	}
	defer sess.Close()
	// FPS works without the network provider, so its failure is only reported.
	netErr := ""
	if err := gamenet.Enable(sess); err != nil {
		netErr = err.Error()
	}

	procErr := make(chan error, 1)
	go func() { procErr <- sess.Process() }()

	var (
		history   []float64
		failures  int
		lastLocal time.Time
		carried   gamenet.SessionFilter
	)
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-procErr:
			if err == nil {
				err = errors.New("ETW session ended unexpectedly")
			}
			postFPS(fps.Status{State: fps.StateError, Process: cfg.Process, Error: err.Error()})
			postNet(gamenet.Status{State: fps.StateError, Error: err.Error()})
			return err
		case <-tick.C:
		}

		if time.Since(lastLocal) >= localRefresh {
			if addrs, err := gamenet.LocalAddrs(); err == nil {
				conns.SetLocal(addrs)
			}
			lastLocal = time.Now()
		}

		st := fps.Status{Process: cfg.Process}
		found, err := fps.FindProcess(cfg.Process)
		if err != nil {
			st.Error = err.Error()
		}
		if found != pid.Load() {
			// Game started, restarted or quit: start a fresh measurement.
			pid.Store(found)
			frames.Reset()
			conns.Reset()
			carried.Reset()
			history = history[:0]
		}
		st.PID = found
		// Steam's traffic only counts while the game runs.
		steam := uint32(0)
		if found != 0 {
			steam, _ = fps.FindProcess(steamProcess)
		}
		steamPID.Store(steam)
		ns := gamenet.Status{PID: found, Error: netErr}
		now := nowFiletime()
		if found == 0 {
			st.State, ns.State = fps.StateWaiting, fps.StateWaiting
		} else {
			st.State = fps.StateRunning
			st.FrameStats = frames.Stats(now)
			history = append(history, st.FPS)
			if len(history) > historyLen {
				history = history[len(history)-historyLen:]
			}
			st.History = history
			ns.State = fps.StateRunning
			if netErr != "" {
				ns.State = fps.StateError
			}
			flows := carried.Filter(conns.Snapshot(now, 2*maxFlows))
			ns.Flows = flows[:min(len(flows), maxFlows)]
		}

		err = postFPS(st)
		if err == nil {
			err = postNet(ns)
		}
		if err != nil {
			if failures++; failures >= 3 {
				return err
			}
		} else {
			failures = 0
		}
	}
}

func nowFiletime() int64 {
	ft := windows.NsecToFiletime(time.Now().UnixNano())
	return int64(ft.HighDateTime)<<32 | int64(ft.LowDateTime)
}
