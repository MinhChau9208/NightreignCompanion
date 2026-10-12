// Package hotkey parses hotkey strings such as "Ctrl+Shift+F1" and, on
// Windows, registers them as system-wide hotkeys (RegisterHotKey). That is
// an ordinary OS API: the app is told about the key press by Windows and
// never touches the game process.
package hotkey

import (
	"errors"
	"fmt"
	"strings"
)

// Modifier flags, equal to the Win32 MOD_* values.
const (
	ModAlt   = 0x1
	ModCtrl  = 0x2
	ModShift = 0x4
	ModWin   = 0x8
)

// Combo is a parsed hotkey: modifier flags plus a Windows virtual-key code.
type Combo struct {
	Mods uint32
	VK   uint32
	name string // canonical key name, e.g. "F1"
}

// String returns the canonical form, modifiers in the order Ctrl, Alt,
// Shift, Win.
func (c Combo) String() string {
	var parts []string
	for _, m := range []struct {
		flag uint32
		name string
	}{{ModCtrl, "Ctrl"}, {ModAlt, "Alt"}, {ModShift, "Shift"}, {ModWin, "Win"}} {
		if c.Mods&m.flag != 0 {
			parts = append(parts, m.name)
		}
	}
	return strings.Join(append(parts, c.name), "+")
}

var modNames = map[string]uint32{
	"ctrl": ModCtrl, "control": ModCtrl,
	"alt":   ModAlt,
	"shift": ModShift,
	"win":   ModWin,
}

type key struct {
	name string
	vk   uint32
}

// keys maps lower-case key names to their canonical name and virtual-key code.
var keys = func() map[string]key {
	m := map[string]key{}
	add := func(name string, vk uint32) { m[strings.ToLower(name)] = key{name, vk} }
	for i := uint32(0); i < 26; i++ {
		add(string(rune('A'+i)), 0x41+i)
	}
	for i := uint32(0); i < 10; i++ {
		add(string(rune('0'+i)), 0x30+i)
		add(fmt.Sprintf("Num%d", i), 0x60+i)
	}
	for i := uint32(1); i <= 24; i++ {
		add(fmt.Sprintf("F%d", i), 0x6F+i)
	}
	for name, vk := range map[string]uint32{
		"Space": 0x20, "PageUp": 0x21, "PageDown": 0x22, "End": 0x23, "Home": 0x24,
		"Left": 0x25, "Up": 0x26, "Right": 0x27, "Down": 0x28,
		"Insert": 0x2D, "Delete": 0x2E, "Pause": 0x13, "ScrollLock": 0x91,
	} {
		add(name, vk)
	}
	return m
}()

// Parse reads a hotkey such as "Ctrl+Shift+F1" (case-insensitive). A
// registered hotkey swallows the key press system-wide, so letters, digits
// and Space need Ctrl, Alt or Win; otherwise typing would break.
func Parse(s string) (Combo, error) {
	parts := strings.Split(s, "+")
	var c Combo
	for i, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if i < len(parts)-1 {
			m, ok := modNames[p]
			if !ok {
				return Combo{}, fmt.Errorf("hotkey %q: unknown modifier %q", s, p)
			}
			if c.Mods&m != 0 {
				return Combo{}, fmt.Errorf("hotkey %q: modifier listed twice", s)
			}
			c.Mods |= m
			continue
		}
		k, ok := keys[p]
		if !ok {
			return Combo{}, fmt.Errorf("hotkey %q: unknown key %q", s, p)
		}
		c.VK, c.name = k.vk, k.name
	}
	typing := c.VK == 0x20 || c.VK >= 0x30 && c.VK <= 0x39 || c.VK >= 0x41 && c.VK <= 0x5A
	if typing && c.Mods&(ModCtrl|ModAlt|ModWin) == 0 {
		return Combo{}, fmt.Errorf("hotkey %q: %s needs Ctrl, Alt or Win", s, c.name)
	}
	return c, nil
}

// Binding names an action and the hotkey that triggers it.
type Binding struct {
	Action string
	Keys   string
}

// ErrUnsupported is returned by Listen on systems without global hotkeys.
var ErrUnsupported = errors.New("global hotkeys are only supported on Windows")
