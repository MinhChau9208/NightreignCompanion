package netmon

import (
	"math"
	"time"
)

// Sample is the outcome of one probe.
type Sample struct {
	RTT  time.Duration
	Lost bool
}

type Thresholds struct {
	PingMs   int
	JitterMs int
	LossPct  float64
}

// Level grades a target against the thresholds.
const (
	LevelUnknown = "unknown" // no data yet
	LevelGood    = "good"
	LevelWarn    = "warn" // above 2/3 of a threshold, or any loss
	LevelBad     = "bad"  // above a threshold
)

// Stats summarises one target over the sliding window.
type Stats struct {
	Target   string  `json:"target"`
	Probe    Probe   `json:"probe"`
	LastMs   float64 `json:"lastMs"`
	LastLost bool    `json:"lastLost"`
	AvgMs    float64 `json:"avgMs"`
	MinMs    float64 `json:"minMs"`
	MaxMs    float64 `json:"maxMs"`
	JitterMs float64 `json:"jitterMs"`
	LossPct  float64 `json:"lossPct"`
	Sent     int     `json:"sent"`
	Received int     `json:"received"`
	// Session totals since monitoring started (not limited to the window).
	TotalSent int `json:"totalSent"`
	TotalLost int `json:"totalLost"`
	// History is RTT in ms per sample, oldest first; -1 marks a lost packet.
	History []float64 `json:"history"`
	Level   string    `json:"level"`
	Error   string    `json:"error,omitempty"`
}

// Compute fills the window statistics from samples (oldest first).
// Jitter is the mean absolute difference between consecutive received
// RTTs, the same idea as RFC 3550 interarrival jitter without smoothing.
func Compute(samples []Sample, th Thresholds) Stats {
	s := Stats{Sent: len(samples), History: make([]float64, len(samples)), Level: LevelUnknown}
	if len(samples) == 0 {
		return s
	}
	var (
		sum, jitterSum float64
		jitterPairs    int
		prev           = -1.0
	)
	s.MinMs = math.Inf(1)
	for i, x := range samples {
		if x.Lost {
			s.History[i] = -1
			prev = -1
			continue
		}
		ms := float64(x.RTT) / float64(time.Millisecond)
		s.History[i] = ms
		s.Received++
		sum += ms
		s.MinMs = math.Min(s.MinMs, ms)
		s.MaxMs = math.Max(s.MaxMs, ms)
		if prev >= 0 {
			jitterSum += math.Abs(ms - prev)
			jitterPairs++
		}
		prev = ms
	}
	last := samples[len(samples)-1]
	s.LastLost = last.Lost
	if !last.Lost {
		s.LastMs = s.History[len(samples)-1]
	}
	s.LossPct = 100 * float64(s.Sent-s.Received) / float64(s.Sent)
	if s.Received == 0 {
		s.MinMs = 0
		s.Level = LevelBad
		return s
	}
	s.AvgMs = sum / float64(s.Received)
	if jitterPairs > 0 {
		s.JitterMs = jitterSum / float64(jitterPairs)
	}
	s.Level = grade(s, th)
	return s
}

func grade(s Stats, th Thresholds) string {
	over := func(v, limit float64) bool { return limit > 0 && v > limit }
	ping, jitter := float64(th.PingMs), float64(th.JitterMs)
	switch {
	case over(s.AvgMs, ping) || over(s.JitterMs, jitter) || over(s.LossPct, th.LossPct):
		return LevelBad
	case over(s.AvgMs, ping*2/3) || over(s.JitterMs, jitter*2/3) || s.LossPct > 0:
		return LevelWarn
	default:
		return LevelGood
	}
}
