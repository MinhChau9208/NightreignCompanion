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
	DLC  bool      `json:"dlc,omitempty"`
}

// RelicEffect is one effect line as found on relics, deep relics, or as a
// dormant power picked up during an expedition. Values are described in
// text (Effect) because many effects are conditional; a future calculator
// will add structured modifiers for the ones that can be summed.
type RelicEffect struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`       // see relicKinds
	Categories []string   `json:"categories"` // see relicCategories
	Name       Localized  `json:"name"`       // in-game wording (English in both)
	InGame     string     `json:"inGame,omitempty"`
	Effect     *Localized `json:"effect,omitempty"` // nil: the name says it all
	Stacking   string     `json:"stacking"`         // with itself; see stackingModes
	Notes      *Localized `json:"notes,omitempty"`
	Character  string     `json:"character,omitempty"` // empty = any character
	Verified   bool       `json:"verified"`
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
	Night      int        `json:"night,omitempty"` // night_boss: 1 or 2
	DLC        bool       `json:"dlc,omitempty"`
	Stats      *BossStats `json:"stats,omitempty"` // nightlord only
	Verified   bool       `json:"verified"`
}

// BossStats holds a Nightlord's fight forms in normal expeditions and as an
// Everdark Sovereign (empty when that version does not exist).
type BossStats struct {
	Normal   []BossForm `json:"normal"`
	Everdark []BossForm `json:"everdark"`
}

// BossForm is one enemy of a Nightlord fight (a phase or a second boss)
// with the stats shared by its game params.
type BossForm struct {
	Name    string    `json:"name"`
	GameIDs []int     `json:"gameIds"`
	Health  []int     `json:"health"` // solo, duo, trio; nil when unknown
	Poise   []float64 `json:"poise"`  // solo, duo, trio
	// Negation is damage absorption in percent per type (see damageTypes):
	// negative means the boss takes extra damage.
	Negation map[string]float64 `json:"negation"`
	// Resist is the status buildup needed to proc (see statusTypes); a
	// status is listed in either Resist or Immune.
	Resist map[string]int `json:"resist"`
	Immune []string       `json:"immune"`
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
	relicKinds      = map[string]bool{"relic": true, "deep": true, "dormant": true}
	stackingModes   = map[string]bool{"yes": true, "no": true, "na": true, "varies": true, "unknown": true}
	relicCategories = map[string]bool{
		"stat": true, "start": true, "exploration": true, "offensive": true,
		"defensive": true, "regen": true, "character": true, "curse": true,
	}
	phaseKinds  = map[string]bool{"explore": true, "shrink": true, "boss": true}
	bossKinds   = map[string]bool{"night_boss": true, "nightlord": true}
	damageTypes = []string{"standard", "strike", "slash", "pierce", "magic", "fire", "lightning", "holy"}
	statusTypes = []string{"poison", "rot", "bleed", "frost", "sleep", "madness", "blight"}
)
