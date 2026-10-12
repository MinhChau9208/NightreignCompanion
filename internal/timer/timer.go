// Package timer runs the day/night cycle of one expedition (docs/SCOPE.md,
// M2). The app cannot read the game's state, so the player drives the timer
// from outside: start when Day 1 begins, then "next" to confirm that a boss
// is down or to re-sync when the circle visibly starts closing.
//
// Timed phases (explore, shrink) advance on their own when their duration
// runs out. Boss phases are untimed and wait for "next".
package timer

import (
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
)

type State string

const (
	Idle    State = "idle"
	Running State = "running"
	Paused  State = "paused"
	Done    State = "done"
)

// SyncGrace is how long after a timed phase began on its own that "next"
// re-anchors that phase to now instead of skipping it. If the timer runs a
// little ahead of the game, the shrink starts on the app first; pressing
// next when the game catches up should line them up, not jump ahead.
const SyncGrace = 30 * time.Second

// Snapshot is the timer as shown in the UI. It is comparable, so callers
// can publish only when something visible changed.
type Snapshot struct {
	Profile  string `json:"profile"`
	Verified bool   `json:"verified"` // durations checked in game
	State    State  `json:"state"`
	// Phase is the index into the profile's phases; -1 while idle and
	// len(phases) once done.
	Phase     int                `json:"phase"`
	PhaseID   string             `json:"phaseId"`
	Kind      string             `json:"kind"`
	Label     gamedata.Localized `json:"label"`
	NextKind  string             `json:"nextKind"` // "" on the last phase
	NextLabel gamedata.Localized `json:"nextLabel"`
	// Duration is the phase length in seconds, 0 for untimed boss phases.
	Duration int `json:"duration"`
	// Remaining counts down to the end of a timed phase (rounded up, so it
	// reads 0 only as the phase ends).
	Remaining int `json:"remaining"`
	// PhaseElapsed and Elapsed exclude time spent paused.
	PhaseElapsed int `json:"phaseElapsed"`
	Elapsed      int `json:"elapsed"`
}

type Timer struct {
	prof gamedata.TimerProfile
	now  func() time.Time

	phase      int
	runStart   time.Time
	phaseStart time.Time
	pausedAt   time.Time // zero unless paused
	doneAt     time.Time
	auto       bool // current phase was entered because the previous ran out
}

// New returns an idle timer for prof. now may be nil (time.Now).
func New(prof gamedata.TimerProfile, now func() time.Time) *Timer {
	if now == nil {
		now = time.Now
	}
	return &Timer{prof: prof, now: now, phase: -1}
}

func (t *Timer) Profile() gamedata.TimerProfile { return t.prof }

func (t *Timer) state() State {
	switch {
	case t.phase < 0:
		return Idle
	case t.phase >= len(t.prof.Phases):
		return Done
	case !t.pausedAt.IsZero():
		return Paused
	}
	return Running
}

// clock is the time the timer runs on: frozen while paused.
func (t *Timer) clock() time.Time {
	if !t.pausedAt.IsZero() {
		return t.pausedAt
	}
	return t.now()
}

// update moves past timed phases that have run out.
func (t *Timer) update() {
	if t.state() != Running {
		return
	}
	now := t.now()
	for t.phase < len(t.prof.Phases) {
		d := time.Duration(t.prof.Phases[t.phase].Seconds) * time.Second
		if d == 0 || now.Sub(t.phaseStart) < d {
			return
		}
		t.phaseStart = t.phaseStart.Add(d)
		t.enter(t.phase+1, t.phaseStart)
		t.auto = true
	}
}

func (t *Timer) enter(phase int, at time.Time) {
	t.phase, t.phaseStart, t.auto = phase, at, false
	if phase >= len(t.prof.Phases) {
		t.doneAt = at
	}
}

// Start begins Day 1 now. It restarts a finished run and is a no-op while
// a run is in progress.
func (t *Timer) Start() {
	t.update()
	if s := t.state(); s == Running || s == Paused {
		return
	}
	now := t.now()
	t.runStart, t.pausedAt = now, time.Time{}
	t.enter(0, now)
}

func (t *Timer) Pause() {
	t.update()
	if t.state() == Running {
		t.pausedAt = t.now()
	}
}

func (t *Timer) Resume() {
	if t.state() != Paused {
		return
	}
	d := t.now().Sub(t.pausedAt)
	t.runStart = t.runStart.Add(d)
	t.phaseStart = t.phaseStart.Add(d)
	t.pausedAt = time.Time{}
}

// Toggle is the start hotkey: start when idle or done, otherwise pause or
// resume.
func (t *Timer) Toggle() {
	t.update()
	switch t.state() {
	case Idle, Done:
		t.Start()
	case Running:
		t.Pause()
	case Paused:
		t.Resume()
	}
}

// Next says "the next phase begins now": a boss is down, or the game has
// reached the next phase. Shortly after a timed phase began on its own it
// re-syncs that phase instead (see SyncGrace). Next also resumes a pause.
func (t *Timer) Next() {
	t.Resume()
	t.update()
	if t.state() != Running {
		return
	}
	now := t.now()
	ph := t.prof.Phases[t.phase]
	if ph.Kind != "boss" && t.auto && now.Sub(t.phaseStart) < SyncGrace {
		t.phaseStart, t.auto = now, false
		return
	}
	t.enter(t.phase+1, now)
}

// Prev undoes a mistaken Next: the previous phase starts over from now.
func (t *Timer) Prev() {
	t.update()
	if t.phase <= 0 {
		return
	}
	t.enter(t.phase-1, t.clock())
}

// Reset returns to idle.
func (t *Timer) Reset() {
	t.phase, t.pausedAt, t.auto = -1, time.Time{}, false
}

// Snapshot reports the timer after moving past phases that ran out.
func (t *Timer) Snapshot() Snapshot {
	t.update()
	s := Snapshot{Profile: t.prof.ID, Verified: t.prof.Verified, State: t.state(), Phase: t.phase}
	switch s.State {
	case Idle:
		if len(t.prof.Phases) > 0 {
			s.NextKind, s.NextLabel = t.prof.Phases[0].Kind, t.prof.Phases[0].Label
		}
		return s
	case Done:
		s.Elapsed = seconds(t.doneAt.Sub(t.runStart))
		return s
	}
	ph := t.prof.Phases[t.phase]
	s.PhaseID, s.Kind, s.Label, s.Duration = ph.ID, ph.Kind, ph.Label, ph.Seconds
	if t.phase+1 < len(t.prof.Phases) {
		next := t.prof.Phases[t.phase+1]
		s.NextKind, s.NextLabel = next.Kind, next.Label
	}
	now := t.clock()
	in := now.Sub(t.phaseStart)
	s.PhaseElapsed = seconds(in)
	s.Elapsed = seconds(now.Sub(t.runStart))
	if ph.Seconds > 0 {
		left := time.Duration(ph.Seconds)*time.Second - in
		s.Remaining = int((left + time.Second - 1) / time.Second)
	}
	return s
}

func seconds(d time.Duration) int { return int(d / time.Second) }

// Alert reports the warning mark (from marks, in seconds) that the
// countdown passed between two snapshots of the same timed phase, or 0.
func Alert(prev, cur Snapshot, marks []int) int {
	if cur.State != Running || prev.State != Running || prev.Phase != cur.Phase || cur.Duration == 0 {
		return 0
	}
	for _, m := range marks {
		if prev.Remaining > m && cur.Remaining <= m && m < cur.Duration {
			return m
		}
	}
	return 0
}
