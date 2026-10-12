package main

import (
	"testing"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
	"github.com/MinhChau9208/NightreignCompanion/internal/store"
)

func TestCheckRun(t *testing.T) {
	d, err := gamedata.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	ok := store.Run{
		Mode: "deep", Depth: 3, Character: "wylder", Result: "win", ClearSeconds: 2400,
		Night1Boss: "demi_human_duo", Night2Boss: "fell_omen", Nightlord: "gladius",
	}
	if err := checkRun(d, ok); err != nil {
		t.Fatalf("valid run rejected: %v", err)
	}
	if err := checkRun(d, store.Run{Mode: "normal", Result: "abandoned"}); err != nil {
		t.Fatalf("minimal run rejected: %v", err)
	}

	bad := map[string]func(*store.Run){
		"mode":          func(r *store.Run) { r.Mode = "ranked" },
		"depth normal":  func(r *store.Run) { r.Mode = "normal" },
		"depth deep":    func(r *store.Run) { r.Depth = 0 },
		"depth too big": func(r *store.Run) { r.Depth = maxDepth + 1 },
		"result":        func(r *store.Run) { r.Result = "draw" },
		"clear time":    func(r *store.Run) { r.ClearSeconds = -1 },
		"character":     func(r *store.Run) { r.Character = "nobody" },
		"nightlord":     func(r *store.Run) { r.Nightlord = "demi_human_duo" },
		"night 1 boss":  func(r *store.Run) { r.Night1Boss = "fell_omen" },
		"night 2 boss":  func(r *store.Run) { r.Night2Boss = "gladius" },
	}
	for name, mutate := range bad {
		r := ok
		mutate(&r)
		if checkRun(d, r) == nil {
			t.Errorf("%s: invalid run accepted: %+v", name, r)
		}
	}
}
