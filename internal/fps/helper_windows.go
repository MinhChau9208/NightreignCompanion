package fps

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"golang.org/x/sys/windows"

	"github.com/MinhChau9208/NightreignCompanion/internal/ipc"
)

// RunHelper is the body of the elevated helper process: it measures the
// target process and reports to the main process until told to stop or
// the main process goes away.
func RunHelper(ctx context.Context, cfg HelperConfig) error {
	if cfg.Window <= 0 {
		cfg.Window = 30 * time.Second
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	post := func(st Status) error { return ipc.Post(ctx, cfg.Addr, cfg.Token, MsgStats, st) }
	if err := post(Status{State: StateStarting, Process: cfg.Process}); err != nil {
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

	var pid atomic.Uint32
	tr := NewTracker(cfg.Window)
	sess, err := StartSession(SessionName, func(p uint32, ts int64) {
		if p != 0 && p == pid.Load() {
			tr.Add(ts)
		}
	})
	if err != nil {
		post(Status{State: StateError, Process: cfg.Process, Error: err.Error()})
		return err
	}
	defer sess.Close()

	procErr := make(chan error, 1)
	go func() { procErr <- sess.Process() }()

	var (
		history  []float64
		failures int
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
			post(Status{State: StateError, Process: cfg.Process, Error: err.Error()})
			return err
		case <-tick.C:
		}

		st := Status{Process: cfg.Process}
		found, err := FindProcess(cfg.Process)
		if err != nil {
			st.Error = err.Error()
		}
		if found != pid.Load() {
			// Game started, restarted or quit: start a fresh measurement.
			pid.Store(found)
			tr.Reset()
			history = history[:0]
		}
		st.PID = found
		if found == 0 {
			st.State = StateWaiting
		} else {
			st.State = StateRunning
			st.FrameStats = tr.Stats(nowFiletime())
			history = append(history, st.FPS)
			if len(history) > historyLen {
				history = history[len(history)-historyLen:]
			}
			st.History = history
		}

		if err := post(st); err != nil {
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
