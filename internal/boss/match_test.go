package boss

import "testing"

func TestMatch(t *testing.T) {
	aliases := []Alias{
		{"demi_human_duo", "Demi-Human Queen Marigga"},
		{"demi_human_duo", "Demi-Human Swordmaster Onze"},
		{"nights_cavalry_duo", "Night's Cavalry"},
		{"tree_sentinel_trio", "Tree Sentinel"},
		{"draconic_sentinel_trio", "Draconic Tree Sentinel"},
		{"wormface", "Wormface"},
	}
	cases := []struct {
		lines []string
		want  string
	}{
		{[]string{"Demi-Human Queen Marigga"}, "demi_human_duo"},
		{[]string{"Dem i-Human Queen Mariqga"}, "demi_human_duo"}, // split word, misread letter
		{[]string{"Nights Cavalry"}, "nights_cavalry_duo"},
		{[]string{"x  Night's  Cavalry  1/2"}, "nights_cavalry_duo"}, // noise around the name
		{[]string{"Draconic Tree Sentinel"}, "draconic_sentinel_trio"},
		{[]string{"Tree Sentinel"}, "tree_sentinel_trio"},
		{[]string{"W0rmface"}, "wormface"},
		{[]string{"Godskin Apostle"}, ""},
		{[]string{"Queen"}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got, score := Match(c.lines, aliases); got != c.want {
			t.Errorf("%q: got %q (%.2f), want %q", c.lines, got, score, c.want)
		}
	}
}
