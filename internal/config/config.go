// Package config loads and persists user settings as JSON in the
// per-user config directory (%AppData%\NightreignCompanion on Windows).
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MinhChau9208/NightreignCompanion/internal/netmon"
)

// AppDirName is the folder created under the OS user config directory.
const AppDirName = "NightreignCompanion"

// Settings is everything the user can configure. New fields must have a
// sensible zero value or be filled in by Defaults, because older config
// files are decoded on top of Defaults.
type Settings struct {
	Language string          `json:"language"` // "vi" | "en"
	Overlay  OverlaySettings `json:"overlay"`
	Hotkeys  Hotkeys         `json:"hotkeys"`
	Network  NetworkSettings `json:"network"`
	FPS      FPSSettings     `json:"fps"`
	Sharing  Sharing         `json:"sharing"`
}

// FPSSettings configures the elevated FPS helper.
type FPSSettings struct {
	Process string `json:"process"` // image name, e.g. nightreign.exe
}

type OverlaySettings struct {
	X       int     `json:"x"`
	Y       int     `json:"y"`
	Opacity float64 `json:"opacity"` // 0.2 – 1.0
}

type Hotkeys struct {
	TimerStart string `json:"timerStart"`
	TimerSync  string `json:"timerSync"`
	TimerReset string `json:"timerReset"`
	Overlay    string `json:"overlay"`
}

type NetworkSettings struct {
	PingTargets []string   `json:"pingTargets"`
	Thresholds  Thresholds `json:"thresholds"`
}

type Thresholds struct {
	PingMs   int     `json:"pingMs"`
	JitterMs int     `json:"jitterMs"`
	LossPct  float64 `json:"lossPct"`
	MinFPS   int     `json:"minFps"`
}

// Sharing holds the community-data consent (decision D4). Everything is
// off until the user explicitly opts in; Asked records that the consent
// screen has been shown at least once.
type Sharing struct {
	Asked      bool `json:"asked"`
	Bosses     bool `json:"bosses"`
	Builds     bool `json:"builds"`
	ClearTimes bool `json:"clearTimes"`
}

// Defaults returns the settings used on first launch.
func Defaults() Settings {
	return Settings{
		Language: "vi",
		Overlay:  OverlaySettings{X: 20, Y: 20, Opacity: 0.85},
		Hotkeys: Hotkeys{
			TimerStart: "Ctrl+Shift+F1",
			TimerSync:  "Ctrl+Shift+F2",
			TimerReset: "Ctrl+Shift+F3",
			Overlay:    "Ctrl+Shift+O",
		},
		Network: NetworkSettings{
			PingTargets: []string{"gateway", "1.1.1.1", "8.8.8.8"},
			Thresholds:  Thresholds{PingMs: 150, JitterMs: 30, LossPct: 2, MinFPS: 50},
		},
		FPS: FPSSettings{Process: "nightreign.exe"},
	}
}

// MaxPingTargets caps how many targets are probed at once (1 probe/s each).
const MaxPingTargets = 6

// Validate reports settings that the app cannot work with.
func (s Settings) Validate() error {
	var errs []error
	if s.Language != "vi" && s.Language != "en" {
		errs = append(errs, fmt.Errorf("language %q: must be vi or en", s.Language))
	}
	if s.Overlay.Opacity < 0.2 || s.Overlay.Opacity > 1 {
		errs = append(errs, fmt.Errorf("overlay opacity %v: must be between 0.2 and 1", s.Overlay.Opacity))
	}
	if n := len(s.Network.PingTargets); n == 0 || n > MaxPingTargets {
		errs = append(errs, fmt.Errorf("ping targets: need 1 to %d, got %d", MaxPingTargets, n))
	}
	seen := map[string]bool{}
	for _, tgt := range s.Network.PingTargets {
		if _, _, _, err := netmon.ParseTarget(tgt); err != nil {
			errs = append(errs, err)
		}
		if seen[tgt] {
			errs = append(errs, fmt.Errorf("ping target %q listed twice", tgt))
		}
		seen[tgt] = true
	}
	if !validExeName(s.FPS.Process) {
		errs = append(errs, fmt.Errorf("fps process %q: must be a file name like nightreign.exe", s.FPS.Process))
	}
	t := s.Network.Thresholds
	if t.PingMs <= 0 || t.JitterMs <= 0 || t.LossPct < 0 || t.MinFPS <= 0 {
		errs = append(errs, errors.New("network thresholds must be positive"))
	}
	return errors.Join(errs...)
}

// validExeName accepts a bare image name. It ends up on the elevated
// helper's command line, so anything beyond [A-Za-z0-9._-] is refused.
func validExeName(name string) bool {
	if len(name) < 5 || len(name) > 64 || !strings.HasSuffix(strings.ToLower(name), ".exe") {
		return false
	}
	for _, r := range name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-'
		if !ok {
			return false
		}
	}
	return true
}

// Store reads and writes Settings to a single JSON file.
type Store struct {
	path string
	mu   sync.Mutex
	cur  Settings
}

// Dir returns the app's config directory, creating it if needed.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, AppDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// Open loads settings from path. A missing file yields Defaults; an
// unreadable or invalid file is an error so the user's data is never
// silently overwritten.
func Open(path string) (*Store, error) {
	s := &Store{path: path, cur: Defaults()}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	cur := Defaults()
	if err := json.Unmarshal(b, &cur); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := cur.Validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", path, err)
	}
	s.cur = cur
	return s, nil
}

// Get returns a copy of the current settings.
func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.cur
	cur.Network.PingTargets = append([]string(nil), s.cur.Network.PingTargets...)
	return cur
}

// Save validates and atomically writes the settings.
func (s *Store) Save(next Settings) error {
	if err := next.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	s.cur = next
	return nil
}
