package main

import (
	"slices"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
)

// Expedition is what is known about the expedition in progress. The main
// process owns it so the predictor tab, the run form, boss OCR and the
// overlay all see the same bosses.
type Expedition struct {
	Night1    string   `json:"night1"`
	Night2    string   `json:"night2"`
	Nightlord string   `json:"nightlord"`
	Exclude   []string `json:"exclude"`
	// Detected names the fields filled in by boss OCR (night1, night2,
	// nightlord) so the UI can mark them.
	Detected []string `json:"detected"`
}

type expState struct {
	mu sync.Mutex
	e  Expedition
}

func (e Expedition) query() boss.Query {
	return boss.Query{Night1: e.Night1, Night2: e.Night2, Exclude: e.Exclude}
}

func (e Expedition) clone() Expedition {
	e.Exclude = append([]string{}, e.Exclude...)
	e.Detected = append([]string{}, e.Detected...)
	return e
}

// slot names an expedition field by its boss phase: 1 and 2 for the night
// bosses, 3 for the Nightlord.
func slotField(slot int) string {
	return [...]string{"", "night1", "night2", "nightlord"}[slot]
}

func (e *Expedition) field(slot int) *string {
	switch slot {
	case 1:
		return &e.Night1
	case 2:
		return &e.Night2
	}
	return &e.Nightlord
}

func (e *Expedition) get(slot int) string     { return *e.field(slot) }
func (e *Expedition) set(slot int, id string) { *e.field(slot) = id }

// setQuery records the bosses the player picked on the predictor. A field
// the player changed is no longer marked as detected.
func (a *App) setQuery(q boss.Query) {
	a.exp.mu.Lock()
	defer a.exp.mu.Unlock()
	e := &a.exp.e
	for slot, id := range map[int]string{1: q.Night1, 2: q.Night2} {
		if e.get(slot) != id {
			e.set(slot, id)
			e.Detected = slices.DeleteFunc(e.Detected, func(f string) bool { return f == slotField(slot) })
		}
	}
	e.Exclude = append([]string{}, q.Exclude...)
}

// detected stores a boss read off the screen, refreshes the prediction
// and tells the main window.
func (a *App) detected(slot int, id string) {
	a.exp.mu.Lock()
	e := &a.exp.e
	e.set(slot, id)
	if f := slotField(slot); !slices.Contains(e.Detected, f) {
		e.Detected = append(e.Detected, f)
	}
	cur := e.clone()
	a.exp.mu.Unlock()

	if slot < 3 {
		if res, err := a.predict(cur.query()); err == nil {
			a.setBossHints(res)
		}
	}
	runtime.EventsEmit(a.ctx, "boss:expedition", cur)
}

// clearExpedition forgets the bosses when a new expedition starts.
func (a *App) clearExpedition() {
	a.exp.mu.Lock()
	a.exp.e = Expedition{}
	a.exp.mu.Unlock()
	a.setBossHints(boss.Result{})
	runtime.EventsEmit(a.ctx, "boss:expedition", Expedition{Exclude: []string{}, Detected: []string{}})
}

func (a *App) expedition() Expedition {
	a.exp.mu.Lock()
	defer a.exp.mu.Unlock()
	return a.exp.e.clone()
}

// GetExpedition returns the bosses known for the expedition in progress.
func (a *App) GetExpedition() (Expedition, error) {
	if err := a.runsErr(); err != nil {
		return Expedition{}, err
	}
	return a.expedition(), nil
}
