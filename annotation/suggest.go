package annotation

import "strings"

// closest returns the candidate most similar to name (compared
// case-insensitively) and true when it is close enough to be a likely typo — a
// Levenshtein distance within a small length-scaled threshold. It powers the
// "did you mean …?" hint on unknown-name diagnostics (§39.3).
func closest(name string, candidates []string) (string, bool) {
	if name == "" || len(candidates) == 0 {
		return "", false
	}
	lname := strings.ToLower(name)
	best := ""
	bestDist := -1
	for _, c := range candidates {
		d := levenshtein(lname, strings.ToLower(c))
		if bestDist == -1 || d < bestDist || (d == bestDist && c < best) {
			best, bestDist = c, d
		}
	}
	threshold := 2
	if len([]rune(name)) < 4 {
		threshold = 1
	}
	if bestDist >= 0 && bestDist <= threshold {
		return best, true
	}
	return "", false
}

// levenshtein returns the edit distance between a and b.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min3(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// didYouMean formats a suggestion suffix, or "" when there is no close match.
func didYouMean(name, prefix string, candidates []string) string {
	if s, ok := closest(name, candidates); ok {
		return "; did you mean " + prefix + s + "?"
	}
	return ""
}
