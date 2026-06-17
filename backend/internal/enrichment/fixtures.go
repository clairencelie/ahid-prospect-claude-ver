// Package enrichment implements the mocked AI enrichment loop (PRD §M7/§8):
// an async worker that reads a static fixtures file and writes unverified
// suggested relationship edges, standing in for a real crawling/AI pipeline.
package enrichment

import (
	"encoding/json"
	"os"
)

// Fixture mirrors one entry in enrichment_fixtures.json. Either Group (plus
// optional Siblings) or LegalName+Relation is set, matching the two example
// shapes in §8: a group/subsidiary suggestion, or a brand-alias suggestion.
type Fixture struct {
	Group      string   `json:"group,omitempty"`
	Siblings   []string `json:"siblings,omitempty"`
	LegalName  string   `json:"legal_name,omitempty"`
	Relation   string   `json:"relation,omitempty"`
	Confidence float64  `json:"confidence"`
	Source     string   `json:"source"`
}

// LoadFixtures reads the fixtures file, keyed by normalized company name.
func LoadFixtures(path string) (map[string]Fixture, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fixtures map[string]Fixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		return nil, err
	}
	return fixtures, nil
}
