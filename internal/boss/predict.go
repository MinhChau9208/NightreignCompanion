// Package boss predicts the hidden Nightlord of a Deep of Night expedition
// from the night bosses met so far (docs/SCOPE.md, M5).
//
// Each Nightlord draws its Night 1 and Night 2 bosses from its own pool
// (data/boss_priors.json). The model assumes a uniform prior over the
// candidate Nightlords and estimates each pool draw from the player's
// recorded runs with add-one smoothing:
//
//	P(n | L, night) = (seen(L, night, n) + 1) / (seen(L, night) + |pool(L, night)|)
//	P(L | n1, n2)   ∝ P(n1 | L, 1) · P(n2 | L, 2)
//
// With no recorded runs this is a uniform draw within each pool. A boss
// outside a Nightlord's pool rules that Nightlord out entirely, and recorded
// bosses outside the pool are ignored.
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
	// Runs is how many recorded runs shaped the pool draws.
	Runs int `json:"runs"`
}

// Sighting is a recorded run whose Nightlord is known; an empty boss means
// that night was not recorded.
type Sighting struct {
	Nightlord string
	Night1    string
	Night2    string
}

// Seen counts recorded night bosses per Nightlord, keeping only bosses in
// that Nightlord's pool. The zero value means no recorded runs.
type Seen struct {
	counts [2]map[string]map[string]int // night-1 -> Nightlord -> boss -> count
	// Runs is the number of sightings that added at least one count.
	Runs int
}

// Observe counts sightings against pools.
func Observe(pools []gamedata.BossPool, sightings []Sighting) Seen {
	s := Seen{counts: [2]map[string]map[string]int{{}, {}}}
	for _, x := range sightings {
		i := slices.IndexFunc(pools, func(p gamedata.BossPool) bool { return p.Nightlord == x.Nightlord })
		if i < 0 {
			continue
		}
		used := false
		for n, b := range []string{x.Night1, x.Night2} {
			if b == "" || !slices.Contains(night(pools[i], n+1), b) {
				continue
			}
			m := s.counts[n][x.Nightlord]
			if m == nil {
				m = map[string]int{}
				s.counts[n][x.Nightlord] = m
			}
			m[b]++
			used = true
		}
		if used {
			s.Runs++
		}
	}
	return s
}

// draw is P(boss | Nightlord, night); 1 when boss is unknown.
func (s Seen) draw(p gamedata.BossPool, n int, boss string) float64 {
	pool := night(p, n)
	if boss == "" {
		return 1
	}
	if !slices.Contains(pool, boss) {
		return 0
	}
	counts := s.counts[n-1][p.Nightlord] // nil-safe: zero Seen has nil maps
	total := 0
	for _, c := range counts {
		total += c
	}
	return float64(counts[boss]+1) / float64(total+len(pool))
}

func night(p gamedata.BossPool, n int) []string {
	if n == 1 {
		return p.Night1
	}
	return p.Night2
}

// Predict ranks the Nightlords in pools given q and the recorded runs.
func Predict(pools []gamedata.BossPool, seen Seen, q Query) Result {
	res := Result{Signal: q.Night1 != "" || q.Night2 != "", Runs: seen.Runs}
	post := make([]float64, len(pools))
	var total float64
	for i, p := range pools {
		if slices.Contains(q.Exclude, p.Nightlord) {
			continue
		}
		post[i] = seen.draw(p, 1, q.Night1) * seen.draw(p, 2, q.Night2)
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
			res.Night1 = seen.chances(pools, post, 1)
		}
		if q.Night2 == "" {
			res.Night2 = seen.chances(pools, post, 2)
		}
	}
	return res
}

// chances marginalises the pool draw over the Nightlord posterior.
func (s Seen) chances(pools []gamedata.BossPool, post []float64, n int) []Chance {
	sum := map[string]float64{}
	var order []string
	for i, p := range pools {
		for _, id := range night(p, n) {
			if _, ok := sum[id]; !ok {
				order = append(order, id)
			}
			sum[id] += post[i] * s.draw(p, n, id)
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
