// Package matching implements the duplicate/group detection engine (PRD §M3):
// name normalization, fuzzy similarity, weighted scoring, and band decisions.
package matching

import "strings"

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
