package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenMissingFileUsesDefaults(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	got := s.Get()
	if got.Language != "vi" {
		t.Errorf("language = %q, want vi", got.Language)
	}
	if got.Sharing.Bosses || got.Sharing.Builds || got.Sharing.ClearTimes || got.Sharing.Asked {
		t.Errorf("sharing must default to off, got %+v", got.Sharing)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	next := s.Get()
	next.Language = "en"
	next.Sharing = Sharing{Asked: true, Bosses: true}
	if err := s.Save(next); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reopened.Get()
	if got.Language != "en" || !got.Sharing.Bosses || got.Sharing.Builds {
		t.Errorf("round trip mismatch: %+v", got)
	}
}

func TestSaveRejectsInvalid(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	bad := s.Get()
	bad.Overlay.Opacity = 0
	if err := s.Save(bad); err == nil {
		t.Fatal("expected error for opacity 0")
	}
	if s.Get().Overlay.Opacity != Defaults().Overlay.Opacity {
		t.Error("invalid save must not change current settings")
	}
}

func TestOpenPartialFileKeepsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"language":"en"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := s.Get()
	if got.Language != "en" || got.Overlay.Opacity != Defaults().Overlay.Opacity {
		t.Errorf("partial file not merged onto defaults: %+v", got)
	}
}

func TestValidatePingTargets(t *testing.T) {
	cases := []struct {
		name    string
		targets []string
		ok      bool
	}{
		{"defaults", Defaults().Network.PingTargets, true},
		{"tcp target", []string{"example.com:443"}, true},
		{"empty list", nil, false},
		{"blank entry", []string{"1.1.1.1", " "}, false},
		{"duplicate", []string{"1.1.1.1", "1.1.1.1"}, false},
		{"too many", []string{"a", "b", "c", "d", "e", "f", "g"}, false},
	}
	for _, tc := range cases {
		s := Defaults()
		s.Network.PingTargets = tc.targets
		if err := s.Validate(); (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}

func TestValidateFPSProcess(t *testing.T) {
	for name, ok := range map[string]bool{
		"nightreign.exe":       true,
		"eldenring_v2-x.EXE":   true,
		"":                     false,
		"nightreign":           false,
		`C:\Games\x.exe`:       false,
		"a b.exe":              false,
		`x.exe" --evil "y.exe`: false,
	} {
		s := Defaults()
		s.FPS.Process = name
		if err := s.Validate(); (err == nil) != ok {
			t.Errorf("%q: err = %v, want ok=%v", name, err, ok)
		}
	}
}
