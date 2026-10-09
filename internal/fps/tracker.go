// Package fps measures a game's frame rate from DXGI Present events read
// through ETW (the same source PresentMon uses). Nothing is injected into
// the game, so it is safe with Easy Anti-Cheat; reading a kernel ETW
// session does require Administrator rights, which is why this runs in a
// separate elevated helper process (docs/SCOPE.md §2.4).
package fps

import (
	"slices"
	"sync"
	"time"
)

// Timestamps are Windows FILETIME ticks (100 ns), as delivered by ETW.
const ticksPerSecond = int64(time.Second / 100)

// stallAfter is how long without a frame before FPS is reported as 0
// (game minimised, loading screen, paused presenting).
const stallAfter = 2 * ticksPerSecond

// Tracker keeps the Present timestamps of one process over a sliding window.
type Tracker struct {
	mu     sync.Mutex
	window int64
	frames []int64 // sorted ascending
}

func NewTracker(window time.Duration) *Tracker {
	return &Tracker{window: int64(window / 100)}
}

// Add records one Present. ETW may deliver events slightly out of order
// across CPU buffers, so late frames are inserted in place.
func (t *Tracker) Add(ts int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n := len(t.frames); n == 0 || ts >= t.frames[n-1] {
		t.frames = append(t.frames, ts)
	} else {
		i, _ := slices.BinarySearch(t.frames, ts)
		t.frames = slices.Insert(t.frames, i, ts)
	}
}

func (t *Tracker) Reset() {
	t.mu.Lock()
	t.frames = t.frames[:0]
	t.mu.Unlock()
}

// FrameStats summarises the window.
type FrameStats struct {
	FPS            float64 `json:"fps"`            // frames in the last second
	AvgFPS         float64 `json:"avgFps"`         // over the whole window
	Low1FPS        float64 `json:"low1Fps"`        // from the slowest 1% of frames
	FrametimeMs    float64 `json:"frametimeMs"`    // mean over the last second
	MaxFrametimeMs float64 `json:"maxFrametimeMs"` // worst frame in the window
	Frames         int     `json:"frames"`
}

// Stats computes the window ending at now. ETW delivers events in batches
// (up to ~1 s late), so "the last second" is measured back from the newest
// frame, not from now; now is only used to drop old frames and detect stalls.
func (t *Tracker) Stats(now int64) FrameStats {
	t.mu.Lock()
	defer t.mu.Unlock()

	cut, _ := slices.BinarySearch(t.frames, now-t.window)
	t.frames = slices.Delete(t.frames, 0, cut)
	f := t.frames
	s := FrameStats{Frames: len(f)}
	if len(f) < 2 {
		return s
	}
	last := f[len(f)-1]

	times := make([]float64, len(f)-1)
	for i := 1; i < len(f); i++ {
		times[i-1] = float64(f[i]-f[i-1]) / float64(ticksPerSecond) * 1000
	}
	span := float64(last-f[0]) / float64(ticksPerSecond)
	s.AvgFPS = float64(len(f)-1) / span
	s.MaxFrametimeMs = slices.Max(times)

	sorted := slices.Clone(times)
	slices.Sort(sorted)
	worst := sorted[len(sorted)-max(1, len(sorted)/100):]
	var sum float64
	for _, v := range worst {
		sum += v
	}
	s.Low1FPS = 1000 / (sum / float64(len(worst)))

	if now-last > stallAfter {
		return s // stalled: FPS and frametime stay 0
	}
	start, _ := slices.BinarySearch(f, last-ticksPerSecond)
	recent := times[max(0, start-1):] // times[i-1] is the frametime ending at frame i
	s.FPS = float64(len(f) - start)
	sum = 0
	for _, v := range recent {
		sum += v
	}
	s.FrametimeMs = sum / float64(len(recent))
	return s
}
