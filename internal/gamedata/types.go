// Package gamedata loads and validates the versioned game data pack
// (relics, timer profiles, bosses) described in docs/SCOPE.md §4.
package gamedata

// Localized is a display string in every supported UI language.
type Localized struct {
	EN string `json:"en"`
	VI string `json:"vi"`
}

type Manifest struct {
	DataVersion string `json:"dataVersion"`
	GameVersion string `json:"gameVersion"`
	Updated     string `json:"updated"`
	Note        string `json:"note"`
}

type Character struct {
	ID   string    `json:"id"`
	Name Localized `json:"name"`
}

// RelicEffect is one relic effect line, e.g. "Physical Attack Up +2".
type RelicEffect struct {
	ID         string      `json:"id"`
	Name       Localized   `json:"name"`
	Category   string      `json:"category"`
	Tiers      []RelicTier `json:"tiers"`
	Stacking   string      `json:"stacking"`   // see stackingModes
	Characters []string    `json:"characters"` // empty = all characters
	DeepOnly   bool        `json:"deepOnly"`
	Negative   bool        `json:"negative"`
	Source     string      `json:"source"`
	Verified   bool        `json:"verified"`
	VerifiedOn string      `json:"verifiedOn,omitempty"`
}

// RelicTier is the effect at a given "+N" level. Value is nil while the
// real number is still unknown.
type RelicTier struct {
	Level int      `json:"level"`
	Value *float64 `json:"value"`
	Unit  string   `json:"unit"` // see tierUnits
}

type TimerProfile struct {
	ID       string    `json:"id"`
	Name     Localized `json:"name"`
	Verified bool      `json:"verified"`
	Note     string    `json:"note,omitempty"`
	Phases   []Phase   `json:"phases"`
}

// Phase is one step of an expedition. Boss phases are untimed (Seconds 0):
// the timer waits for the user to confirm the fight is over.
type Phase struct {
	ID      string    `json:"id"`
	Kind    string    `json:"kind"` // explore | shrink | boss
	Seconds int       `json:"seconds"`
	Label   Localized `json:"label"`
}

type Boss struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"` // night_boss | nightlord
	Name       Localized  `json:"name"`
	Expedition *Localized `json:"expedition,omitempty"`
	Weaknesses []string   `json:"weaknesses"`
	Verified   bool       `json:"verified"`
}

// BossPriors lists which night bosses can appear for each Nightlord; the
// predictor (M5) uses it as the likelihood prior.
type BossPriors struct {
	Verified bool       `json:"verified"`
	Note     string     `json:"note,omitempty"`
	Pools    []BossPool `json:"pools"`
}

type BossPool struct {
	Nightlord string   `json:"nightlord"`
	Night1    []string `json:"night1"`
	Night2    []string `json:"night2"`
}

var (
	stackingModes = map[string]bool{"additive": true, "multiplicative": true, "none": true, "unknown": true}
	tierUnits     = map[string]bool{"multiplier": true, "flat": true, "percent": true}
	phaseKinds    = map[string]bool{"explore": true, "shrink": true, "boss": true}
	bossKinds     = map[string]bool{"night_boss": true, "nightlord": true}
)
