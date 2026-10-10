package netmon

import (
	"context"
	"errors"
	"math"
	"net"
	"sync"
	"testing"
	"time"
)

func ms(n float64) time.Duration { return time.Duration(n * float64(time.Millisecond)) }

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestComputeStats(t *testing.T) {
	th := Thresholds{PingMs: 150, JitterMs: 30, LossPct: 2}
	samples := []Sample{
		{RTT: ms(20)}, {RTT: ms(30)}, {Lost: true}, {RTT: ms(25)}, {RTT: ms(45)},
	}
	s := Compute(samples, th)

	if s.Sent != 5 || s.Received != 4 || !near(s.LossPct, 20) {
		t.Errorf("sent/received/loss = %d/%d/%v", s.Sent, s.Received, s.LossPct)
	}
	if !near(s.AvgMs, 30) || s.MinMs != 20 || s.MaxMs != 45 || s.LastMs != 45 {
		t.Errorf("avg/min/max/last = %v/%v/%v/%v", s.AvgMs, s.MinMs, s.MaxMs, s.LastMs)
	}
	// Pairs across the lost packet are skipped: |30-20| and |45-25|.
	if !near(s.JitterMs, 15) {
		t.Errorf("jitter = %v, want 15", s.JitterMs)
	}
	if s.History[2] != -1 || s.History[0] != 20 {
		t.Errorf("history = %v", s.History)
	}
	if s.Level != LevelBad { // 20% loss > 2%
		t.Errorf("level = %s, want bad", s.Level)
	}
}

func TestComputeLevels(t *testing.T) {
	th := Thresholds{PingMs: 150, JitterMs: 30, LossPct: 2}
	steady := func(rtt float64) []Sample {
		return []Sample{{RTT: ms(rtt)}, {RTT: ms(rtt)}, {RTT: ms(rtt)}}
	}
	cases := []struct {
		name    string
		samples []Sample
		want    string
	}{
		{"no data", nil, LevelUnknown},
		{"all lost", []Sample{{Lost: true}, {Lost: true}}, LevelBad},
		{"fast", steady(30), LevelGood},
		{"over 2/3 of ping", steady(120), LevelWarn},
		{"over ping", steady(200), LevelBad},
		{"jittery", []Sample{{RTT: ms(10)}, {RTT: ms(60)}, {RTT: ms(10)}}, LevelBad},
	}
	for _, tc := range cases {
		if got := Compute(tc.samples, th).Level; got != tc.want {
			t.Errorf("%s: level = %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in, method, host, port string
		ok                     bool
	}{
		{"1.1.1.1", "icmp", "1.1.1.1", "", true},
		{"gateway", "icmp", "gateway", "", true},
		{"example.com:443", "tcp", "example.com", "443", true},
		{"[::1]:80", "tcp", "::1", "80", true},
		{"", "", "", "", false},
		{"bad host", "", "", "", false},
		{":443", "", "", "", false},
	}
	for _, tc := range cases {
		m, h, p, err := ParseTarget(tc.in)
		if (err == nil) != tc.ok || m != tc.method || h != tc.host || p != tc.port {
			t.Errorf("ParseTarget(%q) = %q %q %q %v", tc.in, m, h, p, err)
		}
	}
}

// fakePinger replays a fixed script of results, then times out.
type fakePinger struct {
	mu     sync.Mutex
	script []error
}

func (f *fakePinger) Ping(context.Context, time.Duration) (time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.script) == 0 {
		return 0, ErrTimeout
	}
	err := f.script[0]
	f.script = f.script[1:]
	return ms(10), err
}

func (f *fakePinger) Close() error { return nil }

func TestMonitorRecordsSamplesAndErrors(t *testing.T) {
	boom := errors.New("host unreachable")
	m := NewMonitor(Config{
		Targets:  []string{"a", "b"},
		Interval: 5 * time.Millisecond,
		Window:   3,
		NewPinger: func(_ context.Context, target string) (Pinger, Probe, error) {
			if target == "b" {
				return nil, Probe{}, errors.New("dns failure")
			}
			return &fakePinger{script: []error{nil, nil, boom}}, Probe{Method: "icmp", Address: "10.0.0.1"}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { m.Run(ctx, nil); close(done) }()
	deadline := time.Now().Add(2 * time.Second)
	for m.Snapshot()[0].TotalSent < 5 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done

	snap := m.Snapshot()
	a, b := snap[0], snap[1]
	if a.Sent != 3 {
		t.Errorf("window not capped: sent = %d", a.Sent)
	}
	if a.TotalSent < 5 || a.TotalLost < 3 {
		t.Errorf("totals = %d sent / %d lost", a.TotalSent, a.TotalLost)
	}
	if a.Probe.Address != "10.0.0.1" {
		t.Errorf("probe = %+v", a.Probe)
	}
	if b.Error != "dns failure" || b.Sent != 0 || b.Level != LevelUnknown {
		t.Errorf("failed target = %+v", b)
	}
}

func TestMonitorSetTargets(t *testing.T) {
	var mu sync.Mutex
	closed := map[string]bool{}
	m := NewMonitor(Config{
		Targets:  []string{"a", "b"},
		Interval: 5 * time.Millisecond,
		NewPinger: func(ctx context.Context, target string) (Pinger, Probe, error) {
			return &closingPinger{onClose: func() {
				mu.Lock()
				closed[target] = true
				mu.Unlock()
			}}, Probe{Method: "icmp", Address: target}, nil
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { m.Run(ctx, nil); close(done) }()
	waitFor(t, func() bool { return m.Snapshot()[0].TotalSent >= 2 })

	m.SetTargets([]string{"a", "c", "c"})
	snap := m.Snapshot()
	if len(snap) != 2 || snap[0].Target != "a" || snap[1].Target != "c" {
		t.Fatalf("targets after SetTargets = %+v", snap)
	}
	if snap[0].TotalSent < 2 {
		t.Errorf("kept target lost its history: %+v", snap[0])
	}
	waitFor(t, func() bool { return m.Snapshot()[1].TotalSent >= 1 })
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return closed["b"] })

	m.SetTargets(nil)
	if n := len(m.Snapshot()); n != 0 {
		t.Errorf("%d targets left after clearing", n)
	}
	cancel()
	<-done
}

// closingPinger always answers and reports when it is closed.
type closingPinger struct{ onClose func() }

func (p *closingPinger) Ping(ctx context.Context, _ time.Duration) (time.Duration, error) {
	return ms(5), ctx.Err()
}

func (p *closingPinger) Close() error {
	p.onClose()
	return nil
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met in time")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestTCPPinger(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()

	p, probe, err := NewPinger(context.Background(), ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if probe.Method != "tcp" {
		t.Fatalf("method = %s", probe.Method)
	}
	if _, err := p.Ping(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
}
