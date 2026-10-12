package main

import (
	"sync"

	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
)

// BossData is everything the Boss page needs from the data pack.
type BossData struct {
	Bosses     []gamedata.Boss      `json:"bosses"`
	Pools      []gamedata.BossPool  `json:"pools"`
	Characters []gamedata.Character `json:"characters"` // for the run form
	// PoolsVerified mirrors boss_priors.json: unverified pools must be
	// flagged in the UI.
	PoolsVerified bool `json:"poolsVerified"`
}

func (a *App) GetBossData() (BossData, error) {
	if a.mode != ModeMain {
		return BossData{}, errNotMain
	}
	if a.data == nil {
		return BossData{}, a.initErr
	}
	return BossData{
		Bosses:        a.data.Bosses,
		Pools:         a.data.Priors.Pools,
		Characters:    a.data.Characters,
		PoolsVerified: a.data.Priors.Verified,
	}, nil
}

// PredictNightlord ranks the hidden Nightlord from the night bosses seen,
// learning the pool draws from the player's recorded runs.
func (a *App) PredictNightlord(q boss.Query) (boss.Result, error) {
	if err := a.runsErr(); err != nil {
		return boss.Result{}, err
	}
	s, err := a.sightings()
	if err != nil {
		return boss.Result{}, err
	}
	pools := a.data.Priors.Pools
	res := boss.Predict(pools, boss.Observe(pools, s), q)
	a.setBossHints(res)
	return res, nil
}

// BossHint is a likely Nightlord as shown on the overlay.
type BossHint struct {
	Nightlord   string             `json:"nightlord"`
	Name        gamedata.Localized `json:"name"`
	Probability float64            `json:"probability"`
	// Form is the first fight form of a normal expedition, for the
	// weaknesses; nil when the data pack has no stats.
	Form *gamedata.BossForm `json:"form"`
}

// maxHints is how many Nightlords fit on the overlay.
const maxHints = 2

// bossHints holds the latest prediction for the overlay, re-published
// every second so a newly opened overlay catches up.
type bossHints struct {
	mu   sync.Mutex
	list []BossHint
}

func (a *App) setBossHints(res boss.Result) {
	list := []BossHint{}
	if res.Signal && !res.Impossible {
		for _, c := range res.Ranking {
			if c.Probability == 0 || len(list) == maxHints {
				break
			}
			h := BossHint{Nightlord: c.Nightlord, Probability: c.Probability}
			for _, b := range a.data.Bosses {
				if b.ID == c.Nightlord {
					h.Name = b.Name
					if b.Stats != nil && len(b.Stats.Normal) > 0 {
						h.Form = &b.Stats.Normal[0]
					}
				}
			}
			list = append(list, h)
		}
	}
	a.hints.mu.Lock()
	a.hints.list = list
	a.hints.mu.Unlock()
	a.hub.Publish("boss", list)
}

func (a *App) bossHintList() []BossHint {
	a.hints.mu.Lock()
	defer a.hints.mu.Unlock()
	if a.hints.list == nil {
		return []BossHint{}
	}
	return a.hints.list
}
