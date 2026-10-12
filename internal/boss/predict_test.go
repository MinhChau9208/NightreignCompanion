package boss

import (
	"math"
	"testing"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
)

var testPools = []gamedata.BossPool{
	{Nightlord: "a", Night1: []string{"x", "y"}, Night2: []string{"p"}},
	{Nightlord: "b", Night1: []string{"x", "y", "z", "w"}, Night2: []string{"p", "q"}},
	{Nightlord: "c", Night1: []string{"z"}, Night2: []string{"q"}},
}

func prob(r Result, nightlord string) float64 {
	for _, c := range r.Ranking {
		if c.Nightlord == nightlord {
			return c.Probability
		}
	}
	return -1
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestPredictNoSignalIsUniform(t *testing.T) {
	r := Predict(testPools, Seen{}, Query{})
	if r.Signal || r.Impossible {
		t.Fatalf("signal=%v impossible=%v", r.Signal, r.Impossible)
	}
	for _, c := range r.Ranking {
		if !near(c.Probability, 1.0/3) {
			t.Errorf("%s = %v, want 1/3", c.Nightlord, c.Probability)
		}
	}
}

func TestPredictWeighsPoolSize(t *testing.T) {
	// x: a draws it with 1/2, b with 1/4, c never.
	r := Predict(testPools, Seen{}, Query{Night1: "x"})
	if got := prob(r, "a"); !near(got, 2.0/3) {
		t.Errorf("a = %v, want 2/3", got)
	}
	if got := prob(r, "b"); !near(got, 1.0/3) {
		t.Errorf("b = %v, want 1/3", got)
	}
	if got := prob(r, "c"); got != 0 {
		t.Errorf("c = %v, want 0 (x is not in its pool)", got)
	}
	if r.Ranking[0].Nightlord != "a" {
		t.Errorf("top = %s, want a", r.Ranking[0].Nightlord)
	}
	// Night 2 marginal: p = 2/3·1 + 1/3·1/2, q = 1/3·1/2.
	if len(r.Night2) != 2 || r.Night2[0].Boss != "p" || !near(r.Night2[0].Probability, 5.0/6) {
		t.Errorf("night2 = %+v", r.Night2)
	}
	if r.Night1 != nil {
		t.Errorf("night1 chances given although night 1 is known: %+v", r.Night1)
	}
}

func TestPredictBothNightsAndExclusion(t *testing.T) {
	r := Predict(testPools, Seen{}, Query{Night1: "z", Night2: "q"})
	// b: 1/4·1/2 = 1/8, c: 1·1 = 1.
	if got := prob(r, "c"); !near(got, 8.0/9) {
		t.Errorf("c = %v, want 8/9", got)
	}
	r = Predict(testPools, Seen{}, Query{Night1: "z", Night2: "q", Exclude: []string{"c"}})
	if got := prob(r, "b"); !near(got, 1) {
		t.Errorf("b = %v, want 1 with c excluded", got)
	}
	for _, c := range r.Ranking {
		if c.Excluded != (c.Nightlord == "c") {
			t.Errorf("%s excluded = %v", c.Nightlord, c.Excluded)
		}
	}
}

func TestPredictImpossible(t *testing.T) {
	r := Predict(testPools, Seen{}, Query{Night1: "x", Night2: "q", Exclude: []string{"b"}})
	if !r.Impossible {
		t.Fatalf("want impossible, got %+v", r.Ranking)
	}
	for _, c := range r.Ranking {
		if c.Probability != 0 {
			t.Errorf("%s = %v, want 0", c.Nightlord, c.Probability)
		}
	}
}

func TestPredictEmbeddedPools(t *testing.T) {
	p, err := gamedata.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	// Demi-Human Duo + Fell Omen only fit Gladius and Heolstor, whose pools
	// hold every night boss and so explain them far less well.
	r := Predict(p.Priors.Pools, Seen{}, Query{Night1: "demi_human_duo", Night2: "fell_omen"})
	if r.Ranking[0].Nightlord != "gladius" || r.Ranking[1].Nightlord != "heolstor" || r.Ranking[2].Probability != 0 {
		t.Errorf("ranking = %+v", r.Ranking)
	}
	var sum float64
	for _, c := range r.Ranking {
		sum += c.Probability
	}
	if !near(sum, 1) {
		t.Errorf("probabilities sum to %v", sum)
	}
}

func TestPredictLearnsFromRuns(t *testing.T) {
	seen := Observe(testPools, []Sighting{
		{Nightlord: "a", Night1: "x"},
		{Nightlord: "a", Night1: "x", Night2: "q"}, // q is not in a's pool
		{Nightlord: "c", Night1: "x"},              // x is not in c's pool
		{Nightlord: "nobody", Night1: "x"},
	})
	if seen.Runs != 2 {
		t.Errorf("runs = %d, want 2", seen.Runs)
	}
	// a draws x with (2+1)/(2+2) = 3/4, b with 1/4.
	r := Predict(testPools, seen, Query{Night1: "x"})
	if got := prob(r, "a"); !near(got, 0.75) {
		t.Errorf("a = %v, want 3/4", got)
	}
	if r.Runs != 2 {
		t.Errorf("result runs = %d", r.Runs)
	}
	// With nothing entered, a's Night 1 leans towards x: 1/3·3/4 + 1/3·1/4.
	r = Predict(testPools, seen, Query{})
	for _, c := range r.Night1 {
		if c.Boss == "x" && !near(c.Probability, 1.0/3) {
			t.Errorf("x = %v, want 1/3", c.Probability)
		}
	}
	var sum float64
	for _, c := range r.Night1 {
		sum += c.Probability
	}
	if !near(sum, 1) {
		t.Errorf("night 1 chances sum to %v", sum)
	}
}
