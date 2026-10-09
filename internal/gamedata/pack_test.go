package gamedata

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Chau9208/nightreign-companion/data"
)

func TestEmbeddedPackIsValid(t *testing.T) {
	p, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if p.Stats().Characters == 0 || len(p.Timers) == 0 {
		t.Errorf("embedded pack looks empty: %+v", p.Stats())
	}
}

// packWith returns the embedded pack with one file replaced.
func packWith(t *testing.T, name, content string) fstest.MapFS {
	t.Helper()
	m := fstest.MapFS{}
	entries, err := data.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := data.FS.ReadFile(e.Name())
		if err != nil {
			t.Fatal(err)
		}
		m[e.Name()] = &fstest.MapFile{Data: b}
	}
	m[name] = &fstest.MapFile{Data: []byte(content)}
	return m
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name, file, content, wantErr string
	}{
		{
			"unknown field", "manifest.json",
			`{"dataVersion":"1","bogus":true}`, "unknown field",
		},
		{
			"timed boss phase", "timer_profiles.json",
			`[{"id":"x","name":{"en":"a","vi":"a"},"phases":[{"id":"b","kind":"boss","seconds":10,"label":{"en":"a","vi":"a"}}]}]`,
			"must be untimed",
		},
		{
			"verified relic without value", "relic_effects.json",
			`[{"id":"r","name":{"en":"a","vi":"a"},"category":"attack","stacking":"additive","verified":true,
			   "tiers":[{"level":0,"value":null,"unit":"percent"}]}]`,
			"verified but tier 0 has no value",
		},
		{
			"prior references unknown boss", "boss_priors.json",
			`{"verified":false,"pools":[{"nightlord":"gladius","night1":["nobody"],"night2":[]}]}`,
			`"nobody" is not a night boss`,
		},
		{
			"relic for unknown character", "relic_effects.json",
			`[{"id":"r","name":{"en":"a","vi":"a"},"category":"attack","stacking":"none","characters":["ghost"],
			   "tiers":[{"level":0,"value":null,"unit":"percent"}]}]`,
			`unknown character "ghost"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(packWith(t, tc.file, tc.content))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}
