// Package boss predicts the hidden Nightlord of a Deep of Night expedition
// from the night bosses met so far (docs/SCOPE.md, M5).
//
// Each Nightlord draws its Night 1 and Night 2 bosses from its own pool
// (data/boss_priors.json). With no other information the model assumes a
// uniform prior over the candidate Nightlords and a uniform draw within each
// pool, so
//
//	P(L | n1, n2) ∝ [n1 ∈ pool1(L)]/|pool1(L)| · [n2 ∈ pool2(L)]/|pool2(L)|
//
// A boss outside a Nightlord's pool rules that Nightlord out entirely.
package boss

import (
	"slices"
	"sort"

	"github.com/MinhChau9208/NightreignCompanion/internal/gamedata"
)

// Query is what the player has seen; empty strings mean "not yet".
type Query struct {
	Night1  string   `json:"night1"`
	Night2  string   `json:"night2"`
	Exclude []string `json:"exclude"` // Nightlords the player rules out
}

type Candidate struct {
	Nightlord   string  `json:"nightlord"`
	Probability float64 `json:"probability"`
	Excluded    bool    `json:"excluded"` // by the player, not by the evidence
}

// Chance is the probability that a night boss shows up on a night that has
// not been entered yet.
type Chance struct {
	Boss        string  `json:"boss"`
	Probability float64 `json:"probability"`
}

type Result struct {
	// Ranking lists every pooled Nightlord, most likely first.
	Ranking []Candidate `json:"ranking"`
	// Signal is false when no boss was given, i.e. Ranking is just the prior.
	Signal bool `json:"signal"`
	// Impossible means no candidate's pools contain the given bosses.
	Impossible bool `json:"impossible"`
	// Night1/Night2 are predictions for nights not entered in the query.
	Night1 []Chance `json:"night1"`
	Night2 []Chance `json:"night2"`
}

// Predict ranks the Nightlords in pools given q.
func Predict(pools []gamedata.BossPool, q Query) Result {
	res := Result{Signal: q.Night1 != "" || q.Night2 != ""}
	post := make([]float64, len(pools))
	var total float64
	for i, p := range pools {
		if slices.Contains(q.Exclude, p.Nightlord) {
			continue
		}
		post[i] = draw(p.Night1, q.Night1) * draw(p.Night2, q.Night2)
		total += post[i]
	}
	if total == 0 {
		res.Impossible = true
	}
	for i, p := range pools {
		if total > 0 {
			post[i] /= total
		}
		res.Ranking = append(res.Ranking, Candidate{
			Nightlord:   p.Nightlord,
			Probability: post[i],
			Excluded:    slices.Contains(q.Exclude, p.Nightlord),
		})
	}
	sort.SliceStable(res.Ranking, func(a, b int) bool {
		return res.Ranking[a].Probability > res.Ranking[b].Probability
	})
	if total > 0 {
		if q.Night1 == "" {
			res.Night1 = chances(pools, post, func(p gamedata.BossPool) []string { return p.Night1 })
		}
		if q.Night2 == "" {
			res.Night2 = chances(pools, post, func(p gamedata.BossPool) []string { return p.Night2 })
		}
	}
	return res
}

// draw is P(boss | pool): uniform within the pool, 1 when boss is unknown.
func draw(pool []string, boss string) float64 {
	switch {
	case boss == "":
		return 1
	case slices.Contains(pool, boss):
		return 1 / float64(len(pool))
	default:
		return 0
	}
}

// chances marginalises the pool draw over the Nightlord posterior.
func chances(pools []gamedata.BossPool, post []float64, night func(gamedata.BossPool) []string) []Chance {
	sum := map[string]float64{}
	var order []string
	for i, p := range pools {
		ids := night(p)
		for _, id := range ids {
			if _, seen := sum[id]; !seen {
				order = append(order, id)
			}
			sum[id] += post[i] / float64(len(ids))
		}
	}
	out := make([]Chance, 0, len(order))
	for _, id := range order {
		if sum[id] > 0 {
			out = append(out, Chance{Boss: id, Probability: sum[id]})
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Probability > out[b].Probability })
	return out
}
