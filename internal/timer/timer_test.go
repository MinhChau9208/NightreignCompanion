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
