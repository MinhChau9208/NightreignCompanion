package fps

import (
	"math"
	"testing"
	"time"
)

const ms = ticksPerSecond / 1000

func approx(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

// feed adds n frames spaced by step, starting at start; returns the last ts.
func feed(tr *Tracker, start, step int64, n int) int64 {
	ts := start
	for i := 0; i < n; i++ {
		tr.Add(ts)
		ts += step
	}
	return ts - step
}

func TestSteady60(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	step := ticksPerSecond / 60
	last := feed(tr, 1_000_000, step, 600) // 10 s at 60 fps
	s := tr.Stats(last)

	if !approx(s.AvgFPS, 60, 0.1) || !approx(s.Low1FPS, 60, 0.1) {
		t.Errorf("avg/low1 = %v/%v", s.AvgFPS, s.Low1FPS)
	}
	if !approx(s.FPS, 60, 1) || !approx(s.FrametimeMs, 16.67, 0.05) {
		t.Errorf("fps/frametime = %v/%v", s.FPS, s.FrametimeMs)
	}
}

func TestStutterLowersOnePercentLow(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	step := ticksPerSecond / 60
	ts := int64(0)
	for i := 0; i < 600; i++ {
		if i%100 == 50 {
			ts += 100 * ms // a 100 ms hitch every 100 frames
		} else {
			ts += step
		}
		tr.Add(ts)
	}
	s := tr.Stats(ts)
	if !approx(s.MaxFrametimeMs, 100, 0.01) {
		t.Errorf("max frametime = %v, want 100", s.MaxFrametimeMs)
	}
	// 6 hitches in 599 frametimes: the worst 1% (5 frames) are all hitches.
	if !approx(s.Low1FPS, 10, 0.01) {
		t.Errorf("1%% low = %v, want 10", s.Low1FPS)
	}
	if s.AvgFPS >= 60 || s.AvgFPS < 40 {
		t.Errorf("avg fps = %v", s.AvgFPS)
	}
}

func TestStallReportsZero(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	last := feed(tr, 0, ticksPerSecond/60, 120)
	s := tr.Stats(last + 3*ticksPerSecond)
	if s.FPS != 0 || s.FrametimeMs != 0 {
		t.Errorf("stalled fps/frametime = %v/%v, want 0", s.FPS, s.FrametimeMs)
	}
	if s.AvgFPS == 0 {
		t.Error("window average should survive a short stall")
	}
}

func TestWindowDropsOldFrames(t *testing.T) {
	tr := NewTracker(5 * time.Second)
	feed(tr, 0, ticksPerSecond/30, 300)                         // 10 s at 30 fps
	last := feed(tr, 10*ticksPerSecond, ticksPerSecond/60, 300) // then 5 s at 60
	s := tr.Stats(last)
	if !approx(s.AvgFPS, 60, 0.5) {
		t.Errorf("avg = %v; old 30 fps frames not dropped", s.AvgFPS)
	}
}

func TestOutOfOrderInsert(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	for _, ts := range []int64{0, 2 * ms, 1 * ms, 3 * ms} {
		tr.Add(ts)
	}
	for i := 1; i < len(tr.frames); i++ {
		if tr.frames[i] < tr.frames[i-1] {
			t.Fatalf("frames not sorted: %v", tr.frames)
		}
	}
}

func TestTooFewFrames(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	tr.Add(100)
	if s := tr.Stats(200); s.FPS != 0 || s.Frames != 1 {
		t.Errorf("stats = %+v", s)
	}
}
