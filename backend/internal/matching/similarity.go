package matching

import (
	"sort"
	"strings"
)

// jaroSimilarity returns the Jaro distance metric in [0,1].
func jaroSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1
	}
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)
	if len1 == 0 || len2 == 0 {
		return 0
	}

	matchDistance := max(len1, len2)/2 - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, len1)
	s2Matches := make([]bool, len2)
	matches := 0

	for i := 0; i < len1; i++ {
		start := max(0, i-matchDistance)
		end := min(len2-1, i+matchDistance)
		for j := start; j <= end; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0
	}

	transpositions := 0
	k := 0
	for i := 0; i < len1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}
	transpositions /= 2

	m := float64(matches)
	return (m/float64(len1) + m/float64(len2) + (m-float64(transpositions))/m) / 3
}

// JaroWinkler returns the Jaro-Winkler similarity in [0,1], boosting strings
// that share a common prefix (useful for company-name typo tolerance).
func JaroWinkler(s1, s2 string) float64 {
	jaro := jaroSimilarity(s1, s2)
	r1, r2 := []rune(s1), []rune(s2)
	prefix := 0
	for i := 0; i < len(r1) && i < len(r2) && i < 4; i++ {
		if r1[i] != r2[i] {
			break
		}
		prefix++
	}
	return jaro + float64(prefix)*0.1*(1-jaro)
}

// tokenMatchThreshold is how similar two individual words must be (via
// JaroWinkler) to count as "the same word" with minor typos.
const tokenMatchThreshold = 0.85

// NameSimilarity is the FR3.1 "normalized name similarity" signal: a
// token-set Dice ratio where each word is matched against its best
// remaining counterpart in the other name (greedy, threshold-gated).
//
// Plain whole-string Jaro-Winkler was tried first and rejected: for
// multi-word Indonesian company names, two *unrelated* names routinely
// share enough common letters (vowels, common consonants) to score
// 0.6-0.8 on raw character overlap alone, which falsely inflated the
// over-block guard (PRD FR3.3). Requiring word-level correspondence
// before any credit is given avoids that false-positive class entirely,
// while JaroWinkler at the per-word level still tolerates real typos.
func NameSimilarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}

	tokensA := strings.Fields(a)
	tokensB := strings.Fields(b)
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0
	}

	type pair struct {
		i, j int
		sim  float64
	}
	var candidates []pair
	for i, ta := range tokensA {
		for j, tb := range tokensB {
			if s := JaroWinkler(ta, tb); s >= tokenMatchThreshold {
				candidates = append(candidates, pair{i, j, s})
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].sim > candidates[j].sim })

	usedA := make([]bool, len(tokensA))
	usedB := make([]bool, len(tokensB))
	matchedWeight := 0.0
	for _, p := range candidates {
		if usedA[p.i] || usedB[p.j] {
			continue
		}
		usedA[p.i] = true
		usedB[p.j] = true
		matchedWeight += p.sim
	}

	return 2 * matchedWeight / float64(len(tokensA)+len(tokensB))
}
