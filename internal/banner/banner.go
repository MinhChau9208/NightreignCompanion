// Package banner recognises the banners the game prints mid-screen ("DAY
// II", "The Night's Tide approaches") in OCR'd text, so the cycle timer can
// start and re-sync itself (docs/SCOPE.md, M2).
package banner

import (
	"strings"
	"unicode"

	"github.com/MinhChau9208/NightreignCompanion/internal/boss"
)

// Event kinds.
const (
	Day    = "day"    // Day N begins
	Shrink = "shrink" // a circle shrink begins
)

// Event is one banner seen on screen.
type Event struct {
	Kind string `json:"kind"`
	Day  int    `json:"day,omitempty"` // 1-3 for Day
	Text string `json:"text"`          // the OCR'd line
}

// Message is a banner text and the event kind it marks.
type Message struct {
	Text string
	Kind string
}

// Find returns the first banner in lines: a Day banner, else the message
// that best matches (see boss.Match). ok is false when there is none.
func Find(lines []string, msgs []Message) (ev Event, ok bool) {
	for _, l := range lines {
		if n := ParseDay(l); n > 0 {
			return Event{Kind: Day, Day: n, Text: l}, true
		}
	}
	aliases := make([]boss.Alias, len(msgs))
	for i, m := range msgs {
		aliases[i] = boss.Alias{ID: m.Kind, Name: m.Text}
	}
	kind, _ := boss.Match(lines, aliases)
	if kind == "" {
		return Event{}, false
	}
	return Event{Kind: kind, Text: bestLine(lines, msgs)}, true
}

// bestLine is the line most like any message, for showing what was read.
func bestLine(lines []string, msgs []Message) string {
	var best string
	var score float64
	for _, l := range lines {
		for _, m := range msgs {
			if _, s := boss.Match([]string{l}, []boss.Alias{{ID: "x", Name: m.Text}}); s > score {
				best, score = l, s
			}
		}
	}
	return best
}

// ParseDay reads a Day banner such as "DAY II" and returns the day (1-3),
// or 0. The banner is alone on its line. OCR often reads the Roman
// numeral's strokes as 1, l or |, so each stroke counts as one, and may
// drop the space ("DAYI").
func ParseDay(line string) int {
	s := strings.Join(strings.Fields(strings.TrimFunc(line, func(r rune) bool { return unicode.IsSpace(r) || r == '.' || r == '-' })), "")
	if len(s) < 3 || !strings.EqualFold(s[:3], "day") {
		return 0
	}
	numeral := s[3:]
	if numeral == "" || len(numeral) > 3 {
		return 0
	}
	for _, r := range numeral {
		switch r {
		case 'I', 'i', 'l', '1', '|', '!':
		default:
			return 0
		}
	}
	return len(numeral)
}
