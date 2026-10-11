package main

import (
	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
)

// BossData is everything the Boss page needs from the data pack.
type BossData struct {
	Bosses []gamedata.Boss     `json:"bosses"`
	Pools  []gamedata.BossPool `json:"pools"`
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
	return BossData{Bosses: a.data.Bosses, Pools: a.data.Priors.Pools, PoolsVerified: a.data.Priors.Verified}, nil
}

// PredictNightlord ranks the hidden Nightlord from the night bosses seen.
func (a *App) PredictNightlord(q boss.Query) (boss.Result, error) {
	if a.mode != ModeMain {
		return boss.Result{}, errNotMain
	}
	if a.data == nil {
		return boss.Result{}, a.initErr
	}
	return boss.Predict(a.data.Priors.Pools, q), nil
}
