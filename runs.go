package main

import (
	"fmt"
	"slices"
	"time"

	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
	"github.com/MinhChau9208/NightreignCompanion/internal/store"
)

// maxDepth is the deepest Deep of Night level.
const maxDepth = 5

var runResults = []string{"win", "loss", "abandoned"}

// checkRun rejects runs that reference unknown data or mix up the nights,
// so the stats and the predictor can trust what is stored.
func checkRun(d *gamedata.Pack, r store.Run) error {
	switch r.Mode {
	case "normal":
		if r.Depth != 0 {
			return fmt.Errorf("depth is only for Deep of Night")
		}
	case "deep":
		if r.Depth < 1 || r.Depth > maxDepth {
			return fmt.Errorf("depth must be 1–%d", maxDepth)
		}
	default:
		return fmt.Errorf("unknown mode %q", r.Mode)
	}
	if !slices.Contains(runResults, r.Result) {
		return fmt.Errorf("unknown result %q", r.Result)
	}
	if r.ClearSeconds < 0 {
		return fmt.Errorf("clear time cannot be negative")
	}
	if r.Character != "" && !slices.ContainsFunc(d.Characters, func(c gamedata.Character) bool { return c.ID == r.Character }) {
		return fmt.Errorf("unknown character %q", r.Character)
	}
	if r.Nightlord != "" && !isBoss(d, r.Nightlord, "nightlord", 0) {
		return fmt.Errorf("unknown Nightlord %q", r.Nightlord)
	}
	if r.Night1Boss != "" && !isBoss(d, r.Night1Boss, "night_boss", 1) {
		return fmt.Errorf("%q is not a Night 1 boss", r.Night1Boss)
	}
	if r.Night2Boss != "" && !isBoss(d, r.Night2Boss, "night_boss", 2) {
		return fmt.Errorf("%q is not a Night 2 boss", r.Night2Boss)
	}
	return nil
}

func isBoss(d *gamedata.Pack, id, kind string, night int) bool {
	return slices.ContainsFunc(d.Bosses, func(b gamedata.Boss) bool {
		return b.ID == id && b.Kind == kind && b.Night == night
	})
}

// sightings loads the recorded runs the predictor can learn from.
func (a *App) sightings() ([]boss.Sighting, error) {
	runs, err := a.db.ListRuns(a.ctx, -1)
	if err != nil {
		return nil, err
	}
	var out []boss.Sighting
	for _, r := range runs {
		if r.Nightlord != "" && (r.Night1Boss != "" || r.Night2Boss != "") {
			out = append(out, boss.Sighting{Nightlord: r.Nightlord, Night1: r.Night1Boss, Night2: r.Night2Boss})
		}
	}
	return out, nil
}

// runsErr reports why runs are unavailable.
func (a *App) runsErr() error {
	if a.mode != ModeMain {
		return errNotMain
	}
	if a.db == nil || a.data == nil {
		return a.initErr
	}
	return nil
}

// ---- Bound methods ----

// SaveRun records a finished expedition and returns it with its ID.
func (a *App) SaveRun(r store.Run) (store.Run, error) {
	if err := a.runsErr(); err != nil {
		return store.Run{}, err
	}
	if err := checkRun(a.data, r); err != nil {
		return store.Run{}, err
	}
	// The store assigns ID and time; builds are not recorded yet.
	r.ID, r.CreatedAt, r.BuildJSON = "", time.Time{}, ""
	if err := a.db.InsertRun(a.ctx, &r); err != nil {
		return store.Run{}, err
	}
	// The expedition is over once it is recorded.
	a.clearExpedition()
	return r, nil
}

// ListRuns returns every recorded run, newest first.
func (a *App) ListRuns() ([]store.Run, error) {
	if err := a.runsErr(); err != nil {
		return nil, err
	}
	runs, err := a.db.ListRuns(a.ctx, -1)
	if runs == nil {
		runs = []store.Run{}
	}
	return runs, err
}

func (a *App) DeleteRun(id string) error {
	if err := a.runsErr(); err != nil {
		return err
	}
	return a.db.DeleteRun(a.ctx, id)
}
