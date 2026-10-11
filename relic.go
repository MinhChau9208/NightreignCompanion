package main

import "github.com/MinhChau9208/NightreignCompanion/internal/gamedata"

// RelicData is everything the Relic page needs from the data pack.
type RelicData struct {
	Effects    []gamedata.RelicEffect `json:"effects"`
	Characters []gamedata.Character   `json:"characters"`
}

func (a *App) GetRelicData() (RelicData, error) {
	if a.mode != ModeMain {
		return RelicData{}, errNotMain
	}
	if a.data == nil {
		return RelicData{}, a.initErr
	}
	return RelicData{Effects: a.data.Relics, Characters: a.data.Characters}, nil
}
