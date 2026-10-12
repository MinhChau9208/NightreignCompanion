package main

import (
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
	return boss.Predict(pools, boss.Observe(pools, s), q), nil
}
