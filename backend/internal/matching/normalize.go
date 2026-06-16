// Package matching implements the duplicate/group detection engine (PRD §M3):
// name normalization, fuzzy similarity, weighted scoring, and band decisions.
package matching

import (
	"strings"
	"unicode"
)

// NormalizeDigits strips everything but digits, for comparing NPWP/NIB
// regardless of how the user typed separators (dots, dashes, spaces).
func NormalizeDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// legalForms are stripped as whole tokens (not substrings) per FR3.5, so a
// name like "Sportama" never loses its "PT"-shaped letters.
var legalForms = map[string]bool{
	"PT": true, "CV": true, "PERSERO": true, "TBK": true,
	"PERSEROAN": true, "TERBATAS": true, "LTD": true, "INC": true, "GROUP": true,
}

// NormalizeName implements FR3.5: uppercase, strip legal-form tokens, strip
// punctuation, collapse whitespace. Used for both company names and brand names.
func NormalizeName(raw string) string {
	tokens := tokenize(raw)
	out := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if legalForms[t] {
			continue
		}
		out = append(out, t)
	}
	return strings.Join(out, " ")
}

// NormalizeText applies the same punctuation/case/whitespace normalization
// as NormalizeName but without stripping legal-form tokens — used for
// free-text fields like address where "PT"/"GROUP" aren't legal suffixes.
func NormalizeText(raw string) string {
	return strings.Join(tokenize(raw), " ")
}

func tokenize(raw string) []string {
	upper := strings.ToUpper(raw)
	var b strings.Builder
	for _, r := range upper {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}
