package banner

import "testing"

func TestParseDay(t *testing.T) {
	for line, want := range map[string]int{
		"DAY I":                            1,
		"DAY II":                           2,
		"DAY III":                          3,
		"Day ll":                           2,
		"DAY 11":                           2,
		"DAY I I":                          2,
		" DAY | ":                          1,
		"DAY IIII":                         0,
		"DAY":                              0,
		"DAY 2":                            0,
		"DAYI":                             1,
		"DAYII":                            2,
		"DAY 1":                            1,
		"DAYS":                             0,
		"DAYLIGHT":                         0,
		"DAYS I":                           0,
		"Day 1 – until the rain closes in": 0,
		"Holiday I":                        0,
	} {
		if got := ParseDay(line); got != want {
			t.Errorf("ParseDay(%q) = %d, want %d", line, got, want)
		}
	}
}

func TestFind(t *testing.T) {
	msgs := []Message{
		{Text: "The Night's Tide approaches", Kind: Shrink},
		{Text: "The Night's Tide is shifting", Kind: Shrink},
	}
	cases := []struct {
		lines []string
		want  Event
		ok    bool
	}{
		{[]string{"", "DAY II"}, Event{Kind: Day, Day: 2, Text: "DAY II"}, true},
		{[]string{"The Nights Tlde approaches"}, Event{Kind: Shrink, Text: "The Nights Tlde approaches"}, true},
		{[]string{"Runes 12345", "The Night's Tide is shiftinq"}, Event{Kind: Shrink, Text: "The Night's Tide is shiftinq"}, true},
		{[]string{"The Night approaches"}, Event{}, false},
		{nil, Event{}, false},
	}
	for _, c := range cases {
		got, ok := Find(c.lines, msgs)
		if ok != c.ok || got != c.want {
			t.Errorf("Find(%q) = %+v, %v; want %+v, %v", c.lines, got, ok, c.want, c.ok)
		}
	}
}
