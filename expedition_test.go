package main

import (
	"slices"
	"testing"

	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
)

func TestSetQueryKeepsOnlyUntouchedDetections(t *testing.T) {
	a := &App{}
	a.exp.e = Expedition{Night1: "wormface", Night2: "fell_omen", Detected: []string{"night1", "night2"}}

	// The player corrects Night 2 and rules out a Nightlord.
	a.setQuery(boss.Query{Night1: "wormface", Night2: "great_wyrm", Exclude: []string{"libra"}})
	e := a.expedition()
	if e.Night2 != "great_wyrm" || !slices.Equal(e.Exclude, []string{"libra"}) {
		t.Fatalf("query not stored: %+v", e)
	}
	if !slices.Equal(e.Detected, []string{"night1"}) {
		t.Errorf("detected = %v, want only night1", e.Detected)
	}
}
