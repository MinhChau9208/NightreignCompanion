package boss

import (
	"strings"
	"unicode"
)

// Alias is one name a boss can show on screen.
type Alias struct {
	ID   string
	Name string
}

// MinScore is the similarity an OCR'd line needs to count as a boss name:
// about one misread letter in eight.
const MinScore = 0.86

// Match finds the alias that best matches any of the OCR'd lines. OCR
// misreads letters and splits or merges words, so each alias is compared
// against every run of words of about its length. Ties go to the longer
// alias, so "Draconic Tree Sentinel" beats "Tree Sentinel".
func Match(lines []string, aliases []Alias) (id string, score float64) {
	var bestLen int
	for _, a := range aliases {
		name := normalize(a.Name)
		n := len(strings.Fields(name))
		if n == 0 {
			continue
		}
		for _, line := range lines {
			words := strings.Fields(normalize(line))
			for k := max(1, n-1); k <= n+1; k++ {
				for i := 0; i+k <= len(words); i++ {
					s := similarity(name, strings.Join(words[i:i+k], " "))
					if s > score || (s == score && len(name) > bestLen) {
						id, score, bestLen = a.ID, s, len(name)
					}
				}
			}
		}
	}
	if score < MinScore {
		return "", score
	}
	return id, score
}

// normalize lowercases s, drops apostrophes and turns other punctuation
// into spaces: "Night's Cavalry" and "Demi-Human" become "nights cavalry"
// and "demi human".
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r == '\'' || r == '’':
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// similarity is 1 - edit distance / length of the longer string.
func similarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	n := max(len(ra), len(rb))
	if n == 0 {
		return 1
	}
	return 1 - float64(levenshtein(ra, rb))/float64(n)
}

func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
