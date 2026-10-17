package timer

import (
	"testing"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }
func (c *clock) addSec(s int)        { c.add(time.Duration(s) * time.Second) }
func newClock() *clock               { return &clock{t: time.Date(2026, 10, 11, 20, 0, 0, 0, time.UTC)} }
func phase(id, kind string, s int) gamedata.Phase {
	return gamedata.Phase{ID: id, Kind: kind, Seconds: s, Label: gamedata.Localized{EN: id, VI: id}}
}

var profile = gamedata.TimerProfile{
	ID: "test",
	Phases: []gamedata.Phase{
		phase("explore", "explore", 100),
		phase("shrink", "shrink", 50),
		phase("boss1", "boss", 0),
		phase("day2", "explore", 100),
		phase("boss2", "boss", 0),
	},
}

func check(t *testing.T, s Snapshot, state State, id string, remaining int) {
	t.Helper()
	if s.State != state || s.PhaseID != id || s.Remaining != remaining {
		t.Fatalf("got %s/%q remaining %d, want %s/%q remaining %d", s.State, s.PhaseID, s.Remaining, state, id, remaining)
	}
}

func TestRunThrough(t *testing.T) {
	c := newClock()
	tm := New(profile, c.now)
	s := tm.Snapshot()
	check(t, s, Idle, "", 0)
	if s.NextLabel.EN != "explore" {
		t.Fatalf("idle should preview the first phase, got %q", s.NextLabel.EN)
	}

	tm.Start()
	check(t, tm.Snapshot(), Running, "explore", 100)
	c.add(500 * time.Millisecond)
	check(t, tm.Snapshot(), Running, "explore", 100) // rounds up
	c.add(40500 * time.Millisecond)
	check(t, tm.Snapshot(), Running, "explore", 59)

	// Timed phases chain on their own and the boss phase waits.
	c.addSec(59 + 50 + 600)
	s = tm.Snapshot()
	check(t, s, Running, "boss1", 0)
	if s.PhaseElapsed != 600 || s.Elapsed != 750 {
		t.Fatalf("boss elapsed %d / run %d, want 600 / 750", s.PhaseElapsed, s.Elapsed)
	}

	tm.Next() // boss down
	check(t, tm.Snapshot(), Running, "day2", 100)
	c.addSec(100)
	check(t, tm.Snapshot(), Running, "boss2", 0)
	c.addSec(200)
	tm.Next()
	s = tm.Snapshot()
	if s.State != Done || s.Elapsed != 1050 || s.Phase != len(profile.Phases) {
		t.Fatalf("done: %+v", s)
	}
	c.addSec(100)
	if tm.Snapshot().Elapsed != 1050 {
		t.Fatal("elapsed must stop once done")
	}

	tm.Start() // restart after done
	check(t, tm.Snapshot(), Running, "explore", 100)
	if tm.Snapshot().Elapsed != 0 {
		t.Fatal("restart should reset the run time")
	}
}

func TestPause(t *testing.T) {
	c := newClock()
	tm := New(profile, c.now)
	tm.Start()
	c.addSec(30)
	tm.Toggle() // pause
	c.addSec(1000)
	check(t, tm.Snapshot(), Paused, "explore", 70)
	tm.Toggle() // resume
	check(t, tm.Snapshot(), Running, "explore", 70)
	c.addSec(70)
	s := tm.Snapshot()
	check(t, s, Running, "shrink", 50)
	if s.Elapsed != 100 {
		t.Fatalf("elapsed %d, want 100 (pause excluded)", s.Elapsed)
	}

	c.add(SyncGrace + time.Second) // past the re-sync window
	tm.Pause()
	tm.Next() // resumes and advances
	check(t, tm.Snapshot(), Running, "boss1", 0)
}

func TestNextSync(t *testing.T) {
	c := newClock()
	tm := New(profile, c.now)
	tm.Start()

	// Player says the circle started early: jump to the shrink now.
	c.addSec(90)
	tm.Next()
	check(t, tm.Snapshot(), Running, "shrink", 50)

	// The shrink runs out into the untimed boss phase, which waits.
	c.addSec(60)
	check(t, tm.Snapshot(), Running, "boss1", 0)

	tm.Next() // boss down
	check(t, tm.Snapshot(), Running, "day2", 100)

	// Boss phases always advance on next, even one just entered on its own.
	c.addSec(100)
	check(t, tm.Snapshot(), Running, "boss2", 0)
	tm.Next()
	if tm.Snapshot().State != Done {
		t.Fatal("next on the last boss should finish the run")
	}
}

func TestNextGrace(t *testing.T) {
	c := newClock()
	tm := New(profile, c.now)
	tm.Start()
	c.addSec(105) // shrink began on its own 5s ago
	tm.Next()
	s := tm.Snapshot()
	check(t, s, Running, "shrink", 50) // re-synced, not skipped
	c.addSec(31)
	tm.Next() // a manual entry has no grace
	check(t, tm.Snapshot(), Running, "boss1", 0)

	tm.Prev()
	check(t, tm.Snapshot(), Running, "shrink", 50)
	tm.Reset()
	check(t, tm.Snapshot(), Idle, "", 0)
	tm.Next() // ignored while idle
	check(t, tm.Snapshot(), Idle, "", 0)
}

func TestAlert(t *testing.T) {
	marks := []int{60, 30, 10}
	run := func(phase, remaining int) Snapshot {
		return Snapshot{State: Running, Phase: phase, Duration: 100, Remaining: remaining}
	}
	cases := []struct {
		prev, cur Snapshot
		want      int
	}{
		{run(0, 61), run(0, 60), 60},
		{run(0, 60), run(0, 59), 0},
		{run(0, 32), run(0, 29), 30},
		{run(0, 11), run(0, 10), 10},
		{run(0, 100), run(1, 10), 0}, // phase changed
		{Snapshot{State: Paused, Phase: 0, Duration: 100, Remaining: 61}, run(0, 60), 0},
		{Snapshot{State: Running, Phase: 2, Duration: 0}, Snapshot{State: Running, Phase: 2, Duration: 0}, 0},
	}
	for i, tc := range cases {
		if got := Alert(tc.prev, tc.cur, marks); got != tc.want {
			t.Errorf("case %d: Alert = %d, want %d", i, got, tc.want)
		}
	}
	// A phase shorter than the mark never warns at that mark.
	if got := Alert(Snapshot{State: Running, Duration: 45, Remaining: 61}, Snapshot{State: Running, Duration: 45, Remaining: 60}, marks); got != 0 {
		t.Errorf("short phase: Alert = %d, want 0", got)
	}
}

var dayProfile = gamedata.TimerProfile{
	ID: "days",
	Phases: []gamedata.Phase{
		phase("d1_explore", "explore", 270),
		phase("d1_shrink1", "shrink", 180),
		phase("d1_explore2", "explore", 210),
		phase("d1_shrink2", "shrink", 180),
		phase("night1", "boss", 0),
		phase("d2_explore", "explore", 270),
		phase("d2_shrink1", "shrink", 180),
		phase("night2", "boss", 0),
		phase("nightlord", "boss", 0),
	},
}

func TestDayPhase(t *testing.T) {
	for day, want := range map[int]int{0: -1, 1: 0, 2: 5, 3: 8, 4: -1} {
		if got := DayPhase(dayProfile, day); got != want {
			t.Errorf("DayPhase(%d) = %d, want %d", day, got, want)
		}
	}
}

func TestSync(t *testing.T) {
	c := newClock()
	tm := New(dayProfile, c.now)

	// Day II seen with the timer idle: the run starts there, anchored to
	// when the banner first showed.
	seen := c.t
	c.addSec(2)
	tm.Sync(5, seen)
	s := tm.Snapshot()
	check(t, s, Running, "d2_explore", 268)
	if s.Elapsed != 2 {
		t.Fatalf("elapsed %d, want 2", s.Elapsed)
	}

	// From a boss phase the banner ends the fight; a pause is resumed.
	tm.Sync(7, c.t)
	tm.Pause()
	c.addSec(30)
	tm.Sync(8, c.t.Add(time.Hour)) // future times are clamped
	check(t, tm.Snapshot(), Running, "nightlord", 0)
	if s := tm.Snapshot(); s.PhaseElapsed != 0 {
		t.Fatalf("phase elapsed %d, want 0", s.PhaseElapsed)
	}

	tm.Sync(99, c.t) // out of range: ignored
	check(t, tm.Snapshot(), Running, "nightlord", 0)
}

func TestNearestShrink(t *testing.T) {
	c := newClock()
	tm := New(dayProfile, c.now)
	if _, _, ok := tm.NearestShrink(c.t); ok {
		t.Fatal("idle timer has no shrink")
	}
	tm.Start()

	// The game's first shrink starts 10s before the timer expects it.
	c.addSec(260)
	p, off, ok := tm.NearestShrink(c.t)
	if !ok || p != 1 || off != 10*time.Second {
		t.Fatalf("got %d %v %v, want 1 10s true", p, off, ok)
	}

	// Well into the first gap the second shrink is nearer.
	c.addSec(10 + 180 + 200) // d1_explore2, 10s before the second shrink
	p, off, _ = tm.NearestShrink(c.t)
	if p != 3 || off != 10*time.Second {
		t.Fatalf("got %d %v, want 3 10s", p, off)
	}

	// Timer running ahead: the shrink it already began is 20s early.
	c.addSec(30)
	p, off, _ = tm.NearestShrink(c.t)
	if p != 3 || off != -20*time.Second {
		t.Fatalf("got %d %v, want 3 -20s", p, off)
	}

	// Pauses do not count: the expected start moves with them.
	tm.Pause()
	c.addSec(100)
	p, off, _ = tm.NearestShrink(c.t)
	if p != 3 || off != -20*time.Second {
		t.Fatalf("paused: got %d %v, want 3 -20s", p, off)
	}
	tm.Resume()

	// No shrink during a boss fight.
	c.addSec(200)
	if _, _, ok := tm.NearestShrink(c.t); ok {
		t.Fatal("boss phase has no shrink")
	}
}
