package main

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/MinhChau9208/NightreignCompanion/internal/banner"
	"github.com/MinhChau9208/NightreignCompanion/internal/capture"
	"github.com/MinhChau9208/NightreignCompanion/internal/fps"
	"github.com/MinhChau9208/NightreignCompanion/internal/timer"
)

// Timer sync reads the Day banner in the middle of the screen ("DAY II")
// and the messages at the top right ("The Night's tide grows stronger"),
// and starts or re-syncs the cycle timer. Like
// boss OCR it only captures screen pixels of the game window, and only
// while the game is the window in front.

const (
	// bannerRepeat is how long the same banner is ignored after it synced
	// the timer: banners stay up a few seconds and OCR can lose them for a
	// frame in between.
	bannerRepeat = 30 * time.Second
	// maxShrinkOff is how far the timer may be off from a shrink banner and
	// still be corrected; further off, the banner is likely a misread or a
	// message the data pack maps wrongly.
	maxShrinkOff = 2 * time.Minute
)

// Timer sync states shown in the UI.
const (
	syncOff        = "off"        // disabled in settings
	syncWatching   = "watching"   // looking for banners
	syncBackground = "background" // the game is not the window in front
	syncNoGame     = "noGame"     // game process or window not found
	syncError      = "error"
)

// SyncEvent is a banner that moved the timer.
type SyncEvent struct {
	Kind  string `json:"kind"` // banner.Day or banner.Shrink
	Day   int    `json:"day"`
	Text  string `json:"text"`  // as read
	Phase string `json:"phase"` // phase id the timer moved to
	At    int64  `json:"at"`    // unix ms when the banner first showed
}

type SyncStatus struct {
	State string    `json:"state"`
	Error string    `json:"error"`
	Last  SyncEvent `json:"last"` // zero At: none yet
}

type bannerScanner struct {
	mu     sync.Mutex
	status SyncStatus
	// cand is the banner seen in the last captures, how many times, and
	// when it first showed.
	cand   banner.Event
	streak int
	first  time.Time
	// applied is the last banner that synced the timer.
	applied   banner.Event
	appliedAt time.Time
}

func (a *App) setSyncStatus(st SyncStatus) {
	b := &a.banner
	b.mu.Lock()
	if st.Last.At == 0 {
		st.Last = b.status.Last
	}
	changed := b.status != st
	b.status = st
	if st.State != syncWatching {
		b.cand, b.streak = banner.Event{}, 0
	}
	b.mu.Unlock()
	if changed {
		runtime.EventsEmit(a.ctx, "timer:sync", st)
	}
}

func (a *App) scanBanner() SyncStatus {
	cfg := a.cfg.Get()
	if !cfg.OCR.TimerSync {
		return SyncStatus{State: syncOff}
	}
	pid, err := fps.FindProcess(cfg.FPS.Process)
	if err != nil {
		return SyncStatus{State: syncError, Error: err.Error()}
	}
	if pid == 0 {
		return SyncStatus{State: syncNoGame}
	}
	// Otherwise the capture would read whatever covers the game.
	if !capture.Foreground(pid) {
		return SyncStatus{State: syncBackground}
	}
	win, err := capture.Window(pid)
	if errors.Is(err, capture.ErrNoWindow) {
		return SyncStatus{State: syncNoGame}
	}
	if err != nil {
		return SyncStatus{State: syncError, Error: err.Error()}
	}
	seen := time.Now()
	var lines []string
	for _, r := range []capture.Region{cfg.OCR.DayRegion, cfg.OCR.MessageRegion} {
		img, err := capture.Grab(r.In(win))
		if err != nil {
			return SyncStatus{State: syncError, Error: err.Error()}
		}
		l, err := a.readText(img)
		if err != nil {
			return SyncStatus{State: syncError, Error: err.Error()}
		}
		lines = append(lines, l...)
	}
	ev, ok := banner.Find(lines, a.messages())

	b := &a.banner
	b.mu.Lock()
	switch {
	case !ok:
		b.cand, b.streak = banner.Event{}, 0
	case sameBanner(ev, b.cand):
		b.streak++
	default:
		b.cand, b.streak, b.first = ev, 1, seen
	}
	// Act once per showing, when it is confirmed.
	confirmed := ok && b.streak == ocrConfirm
	at := b.first
	if confirmed && sameBanner(ev, b.applied) && at.Sub(b.appliedAt) < bannerRepeat {
		confirmed = false
	}
	b.mu.Unlock()

	st := SyncStatus{State: syncWatching}
	if confirmed {
		if phase := a.syncFromBanner(ev, at); phase != "" {
			b.mu.Lock()
			b.applied, b.appliedAt = ev, at
			b.mu.Unlock()
			st.Last = SyncEvent{Kind: ev.Kind, Day: ev.Day, Text: ev.Text, Phase: phase, At: at.UnixMilli()}
		}
	}
	return st
}

func sameBanner(a, b banner.Event) bool { return a.Kind == b.Kind && a.Day == b.Day }

// messages lists the non-Day banners from the data pack.
func (a *App) messages() []banner.Message {
	out := make([]banner.Message, len(a.data.Messages.Messages))
	for i, m := range a.data.Messages.Messages {
		out[i] = banner.Message{Text: m.Text, Kind: m.Event}
	}
	return out
}

// syncFromBanner moves the timer to the phase a banner marks, as of at,
// and returns that phase's id, or "" when the banner changed nothing.
//
//   - Day N jumps to the first phase of that day, which also ends the
//     night boss fight before it. Day I starts a new run; if a run was in
//     progress or finished, it is a new expedition.
//   - A shrink banner snaps the nearest shrink of the current day to at.
func (a *App) syncFromBanner(ev banner.Event, at time.Time) string {
	c := &a.tmr
	c.mu.Lock()
	newRun := false
	s := c.t.Snapshot()
	switch ev.Kind {
	case banner.Day:
		target := timer.DayPhase(c.t.Profile(), ev.Day)
		if target < 0 {
			c.mu.Unlock()
			return ""
		}
		active := s.State == timer.Running || s.State == timer.Paused
		newRun = s.State == timer.Done || ev.Day == 1 && active && s.Phase > 0
		if ev.Day == 1 || s.State == timer.Done {
			// Pick up a profile chosen since, as reset does.
			c.t = timer.New(a.profile(a.cfg.Get().Timer.Profile), nil)
			if target = timer.DayPhase(c.t.Profile(), ev.Day); target < 0 {
				target = 0
			}
		}
		c.t.Sync(target, at)
	case banner.Shrink:
		phase, off, ok := c.t.NearestShrink(at)
		if !ok || off.Abs() > maxShrinkOff || off.Abs() < time.Second {
			c.mu.Unlock()
			return ""
		}
		c.t.Sync(phase, at)
	default:
		c.mu.Unlock()
		return ""
	}
	s = c.t.Snapshot()
	c.last, c.pub = s, time.Now()
	c.mu.Unlock()

	if newRun {
		a.clearExpedition()
	}
	log.Printf("timer sync: %q -> %s", ev.Text, s.PhaseID)
	a.publishTimer(s)
	return s.PhaseID
}

// ---- Bound methods ----

func (a *App) TimerSyncStatus() SyncStatus {
	a.banner.mu.Lock()
	defer a.banner.mu.Unlock()
	return a.banner.status
}

// TestBannerOCR captures region now and reports what OCR reads there and
// the banner it recognises, if any.
func (a *App) TestBannerOCR(region capture.Region) (OCRTest, error) {
	res, lines, err := a.testCapture(region)
	if err != nil {
		return res, err
	}
	if ev, ok := banner.Find(lines, a.messages()); ok {
		res.Match, res.Day, res.Score = ev.Kind, ev.Day, 1
	}
	return res, nil
}
