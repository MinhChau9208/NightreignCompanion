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
		if !stackingModes[r.Stacking] {
			add("%s: unknown stacking %q", where, r.Stacking)
		}
		if len(r.Tiers) == 0 {
			add("%s: needs at least one tier", where)
		}
		for i, t := range r.Tiers {
			if t.Level != i {
				add("%s: tier %d has level %d, levels must be 0,1,2,…", where, i, t.Level)
			}
			if !tierUnits[t.Unit] {
				add("%s: tier %d unknown unit %q", where, i, t.Unit)
			}
			if r.Verified && t.Value == nil {
				add("%s: verified but tier %d has no value", where, i)
			}
		}
		for _, c := range r.Characters {
			if !chars[c] {
				add("%s: unknown character %q", where, c)
			}
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
	for _, b := range p.Bosses {
		if _, dup := bossKind[b.ID]; dup || b.ID == "" {
			add("bosses: missing or duplicate id %q", b.ID)
		}
		bossKind[b.ID] = b.Kind
		if !bossKinds[b.Kind] {
			add("boss %s: unknown kind %q", b.ID, b.Kind)
		}
		checkName(add, "boss "+b.ID, b.Name)
	}

	for _, pool := range p.Priors.Pools {
		if bossKind[pool.Nightlord] != "nightlord" {
			add("priors: %q is not a nightlord", pool.Nightlord)
		}
		for _, id := range append(append([]string(nil), pool.Night1...), pool.Night2...) {
			if bossKind[id] != "night_boss" {
				add("priors %s: %q is not a night boss", pool.Nightlord, id)
			}
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

func checkName(add func(string, ...any), where string, n Localized) {
	if n.EN == "" || n.VI == "" {
		add("%s: name needs both en and vi", where)
	}
}
