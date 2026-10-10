package netmon

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Config struct {
	Targets    []string
	Interval   time.Duration // time between probes, per target
	Timeout    time.Duration // a probe slower than this counts as lost
	Window     int           // samples kept for the sliding-window stats
	Thresholds Thresholds
	// NewPinger defaults to the package's NewPinger; tests replace it.
	NewPinger func(ctx context.Context, target string) (Pinger, Probe, error)
}

func (c *Config) setDefaults() {
	if c.Interval <= 0 {
		c.Interval = time.Second
	}
	if c.Timeout <= 0 {
		c.Timeout = time.Second
	}
	if c.Window <= 0 {
		c.Window = 60
	}
	if c.NewPinger == nil {
		c.NewPinger = NewPinger
	}
}

// Monitor probes every target concurrently and keeps a sliding window of
// samples per target.
type Monitor struct {
	cfg    Config
	mu     sync.Mutex
	states []*targetState
	runCtx context.Context // set once Run starts
	wg     sync.WaitGroup
}

type targetState struct {
	target    string
	probe     Probe
	err       string
	samples   []Sample // oldest first, at most cfg.Window
	totalSent int
	totalLost int
	cancel    context.CancelFunc // stops this target's probe loop
}

func NewMonitor(cfg Config) *Monitor {
	cfg.setDefaults()
	m := &Monitor{cfg: cfg}
	m.SetTargets(cfg.Targets)
	return m
}

// SetTargets replaces the target list. Targets that stay keep their
// history; removed ones stop probing. Safe to call while Run is active.
func (m *Monitor) SetTargets(targets []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	old := make(map[string]*targetState, len(m.states))
	for _, st := range m.states {
		old[st.target] = st
	}
	next := make([]*targetState, 0, len(targets))
	seen := make(map[string]bool, len(targets))
	for _, t := range targets {
		if seen[t] {
			continue
		}
		seen[t] = true
		if st, ok := old[t]; ok {
			next = append(next, st)
			delete(old, t)
			continue
		}
		st := &targetState{target: t}
		next = append(next, st)
		if m.runCtx != nil && m.runCtx.Err() == nil {
			m.startLocked(st)
		}
	}
	for _, st := range old {
		if st.cancel != nil {
			st.cancel()
		}
	}
	m.states = next
}

func (m *Monitor) startLocked(st *targetState) {
	ctx, cancel := context.WithCancel(m.runCtx)
	st.cancel = cancel
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.probeLoop(ctx, st)
	}()
}

// Run probes until ctx is cancelled, calling onUpdate with a fresh
// snapshot once per interval.
func (m *Monitor) Run(ctx context.Context, onUpdate func([]Stats)) {
	m.mu.Lock()
	m.runCtx = ctx
	for _, st := range m.states {
		m.startLocked(st)
	}
	m.mu.Unlock()
	t := time.NewTicker(m.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			m.wg.Wait()
			return
		case <-t.C:
			if onUpdate != nil {
				onUpdate(m.Snapshot())
			}
		}
	}
}

// retryDelay is how long to wait before re-creating a pinger that failed
// to start (DNS failure, no gateway yet, …).
const retryDelay = 5 * time.Second

func (m *Monitor) probeLoop(ctx context.Context, st *targetState) {
	for ctx.Err() == nil {
		p, probe, err := m.cfg.NewPinger(ctx, st.target)
		if err != nil {
			m.setErr(st, err)
			sleep(ctx, retryDelay)
			continue
		}
		m.mu.Lock()
		st.probe, st.err = probe, ""
		m.mu.Unlock()
		m.probeWith(ctx, st, p)
		p.Close()
	}
}

func (m *Monitor) probeWith(ctx context.Context, st *targetState, p Pinger) {
	t := time.NewTicker(m.cfg.Interval)
	defer t.Stop()
	for {
		rtt, err := p.Ping(ctx, m.cfg.Timeout)
		if ctx.Err() != nil {
			return
		}
		lost := err != nil
		m.mu.Lock()
		st.samples = append(st.samples, Sample{RTT: rtt, Lost: lost})
		if len(st.samples) > m.cfg.Window {
			st.samples = st.samples[len(st.samples)-m.cfg.Window:]
		}
		st.totalSent++
		if lost {
			st.totalLost++
		}
		// Timeouts are ordinary packet loss; anything else is worth showing.
		st.err = ""
		if err != nil && !errors.Is(err, ErrTimeout) {
			st.err = err.Error()
		}
		m.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (m *Monitor) setErr(st *targetState, err error) {
	m.mu.Lock()
	st.err = err.Error()
	m.mu.Unlock()
}

// Snapshot returns the current stats for every target, in config order.
func (m *Monitor) Snapshot() []Stats {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Stats, len(m.states))
	for i, st := range m.states {
		s := Compute(st.samples, m.cfg.Thresholds)
		s.Target, s.Probe, s.Error = st.target, st.probe, st.err
		s.TotalSent, s.TotalLost = st.totalSent, st.totalLost
		out[i] = s
	}
	return out
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
