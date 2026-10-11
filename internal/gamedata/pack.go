package gamedata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"github.com/MinhChau9208/NightreignCompanion/data"
)

// Pack is a fully loaded and validated data pack.
type Pack struct {
	Manifest   Manifest
	Characters []Character
	Relics     []RelicEffect
	Timers     []TimerProfile
	Bosses     []Boss
	Priors     BossPriors
}

// LoadEmbedded loads the data pack bundled into the binary.
func LoadEmbedded() (*Pack, error) { return Load(data.FS) }

// Load reads every pack file from fsys and validates the result.
func Load(fsys fs.FS) (*Pack, error) {
	var p Pack
	files := []struct {
		name string
		dst  any
	}{
		{"manifest.json", &p.Manifest},
		{"characters.json", &p.Characters},
		{"relic_effects.json", &p.Relics},
		{"timer_profiles.json", &p.Timers},
		{"bosses.json", &p.Bosses},
		{"boss_priors.json", &p.Priors},
	}
	for _, f := range files {
		if err := decodeFile(fsys, f.name, f.dst); err != nil {
			return nil, err
		}
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func decodeFile(fsys fs.FS, name string, dst any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Validate checks ids, enums and cross references across the pack.
func (p *Pack) Validate() error {
	var errs []error
	add := func(format string, args ...any) { errs = append(errs, fmt.Errorf(format, args...)) }

	if p.Manifest.DataVersion == "" {
		add("manifest: dataVersion is required")
	}

	chars := map[string]bool{}
	for _, c := range p.Characters {
		if !uniqueID(chars, c.ID) {
			add("characters: missing or duplicate id %q", c.ID)
		}
		checkName(add, "character "+c.ID, c.Name)
	}

	relics := map[string]bool{}
	for _, r := range p.Relics {
		where := "relic " + r.ID
		if !uniqueID(relics, r.ID) {
			add("relics: missing or duplicate id %q", r.ID)
		}
		checkName(add, where, r.Name)
		if !relicKinds[r.Kind] {
			add("%s: unknown kind %q", where, r.Kind)
		}
		if len(r.Categories) == 0 {
			add("%s: needs at least one category", where)
		}
		for _, c := range r.Categories {
			if !relicCategories[c] {
				add("%s: unknown category %q", where, c)
			}
		}
		if !stackingModes[r.Stacking] {
			add("%s: unknown stacking %q", where, r.Stacking)
		}
		if r.Effect != nil {
			checkText(add, where+" effect", *r.Effect)
		}
		if r.Notes != nil {
			checkText(add, where+" notes", *r.Notes)
		}
		if r.Character != "" && !chars[r.Character] {
			add("%s: unknown character %q", where, r.Character)
		}
	}

	timers := map[string]bool{}
	for _, tp := range p.Timers {
		where := "timer " + tp.ID
		if !uniqueID(timers, tp.ID) {
			add("timers: missing or duplicate id %q", tp.ID)
		}
		phases := map[string]bool{}
		for _, ph := range tp.Phases {
			if !uniqueID(phases, ph.ID) {
				add("%s: missing or duplicate phase id %q", where, ph.ID)
			}
			if !phaseKinds[ph.Kind] {
				add("%s: phase %s unknown kind %q", where, ph.ID, ph.Kind)
			}
			switch {
			case ph.Kind == "boss" && ph.Seconds != 0:
				add("%s: boss phase %s must be untimed (seconds 0)", where, ph.ID)
			case ph.Kind != "boss" && ph.Seconds <= 0:
				add("%s: phase %s needs a positive duration", where, ph.ID)
			}
		}
		if len(tp.Phases) == 0 {
			add("%s: has no phases", where)
		}
	}

	bossKind := map[string]string{}
	bossNight := map[string]int{}
	for _, b := range p.Bosses {
		where := "boss " + b.ID
		if _, dup := bossKind[b.ID]; dup || b.ID == "" {
			add("bosses: missing or duplicate id %q", b.ID)
		}
		bossKind[b.ID] = b.Kind
		bossNight[b.ID] = b.Night
		if !bossKinds[b.Kind] {
			add("%s: unknown kind %q", where, b.Kind)
		}
		checkName(add, where, b.Name)
		switch b.Kind {
		case "night_boss":
			if b.Night != 1 && b.Night != 2 {
				add("%s: night must be 1 or 2, got %d", where, b.Night)
			}
			if b.Stats != nil {
				add("%s: only nightlords have stats", where)
			}
		case "nightlord":
			if b.Night != 0 {
				add("%s: nightlords have no night", where)
			}
			if b.Stats != nil {
				checkForms(add, where+" normal", b.Stats.Normal)
				checkForms(add, where+" everdark", b.Stats.Everdark)
			}
		}
	}

	pooled := map[string]bool{}
	for _, pool := range p.Priors.Pools {
		if bossKind[pool.Nightlord] != "nightlord" {
			add("priors: %q is not a nightlord", pool.Nightlord)
		}
		if !uniqueID(pooled, pool.Nightlord) {
			add("priors: duplicate pool for %q", pool.Nightlord)
		}
		for night, ids := range [][]string{pool.Night1, pool.Night2} {
			for _, id := range ids {
				if bossKind[id] != "night_boss" || bossNight[id] != night+1 {
					add("priors %s: %q is not a night %d boss", pool.Nightlord, id, night+1)
				}
			}
		}
		if p.Priors.Verified && (len(pool.Night1) == 0 || len(pool.Night2) == 0) {
			add("priors %s: verified pools need night 1 and night 2 bosses", pool.Nightlord)
		}
	}

	return errors.Join(errs...)
}

// Stats summarises how much of the pack has been verified.
type Stats struct {
	DataVersion    string `json:"dataVersion"`
	GameVersion    string `json:"gameVersion"`
	Characters     int    `json:"characters"`
	Relics         int    `json:"relics"`
	RelicsVerified int    `json:"relicsVerified"`
	Bosses         int    `json:"bosses"`
	BossesVerified int    `json:"bossesVerified"`
	TimerProfiles  int    `json:"timerProfiles"`
	TimersVerified int    `json:"timersVerified"`
	PriorsVerified bool   `json:"priorsVerified"`
}

func (p *Pack) Stats() Stats {
	s := Stats{
		DataVersion:    p.Manifest.DataVersion,
		GameVersion:    p.Manifest.GameVersion,
		Characters:     len(p.Characters),
		Relics:         len(p.Relics),
		Bosses:         len(p.Bosses),
		TimerProfiles:  len(p.Timers),
		PriorsVerified: p.Priors.Verified,
	}
	for _, r := range p.Relics {
		if r.Verified {
			s.RelicsVerified++
		}
	}
	for _, b := range p.Bosses {
		if b.Verified {
			s.BossesVerified++
		}
	}
	for _, t := range p.Timers {
		if t.Verified {
			s.TimersVerified++
		}
	}
	return s
}

func uniqueID(seen map[string]bool, id string) bool {
	if id == "" || seen[id] {
		return false
	}
	seen[id] = true
	return true
}

func checkForms(add func(string, ...any), where string, forms []BossForm) {
	for i, f := range forms {
		at := fmt.Sprintf("%s form %d", where, i)
		if f.Name == "" {
			add("%s: needs a name", at)
		}
		if len(f.GameIDs) == 0 {
			add("%s: needs at least one game id", at)
		}
		if f.Health != nil && len(f.Health) != 3 {
			add("%s: health needs solo, duo and trio values", at)
		}
		if len(f.Poise) != 3 {
			add("%s: poise needs solo, duo and trio values", at)
		}
		if len(f.Negation) != len(damageTypes) {
			add("%s: negation needs exactly %v", at, damageTypes)
		}
		for _, d := range damageTypes {
			if _, ok := f.Negation[d]; !ok {
				add("%s: missing %s negation", at, d)
			}
		}
		immune := map[string]bool{}
		for _, s := range f.Immune {
			immune[s] = true
		}
		if len(f.Resist)+len(f.Immune) != len(statusTypes) {
			add("%s: resist + immune need exactly %v", at, statusTypes)
		}
		for _, s := range statusTypes {
			if _, ok := f.Resist[s]; ok == immune[s] {
				add("%s: status %s must be in exactly one of resist/immune", at, s)
			}
		}
	}
}

func checkName(add func(string, ...any), where string, n Localized) {
	checkText(add, where+" name", n)
}

func checkText(add func(string, ...any), where string, n Localized) {
	if n.EN == "" || n.VI == "" {
		add("%s: needs both en and vi", where)
	}
}
