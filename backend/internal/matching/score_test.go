package matching

import (
	"testing"

	"ahid-prospect/backend/internal/domain"
)

var defaultThresholds = Thresholds{BlockThreshold: 80, ReviewThreshold: 50}

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"PT Delta Giri Wacana":        "DELTA GIRI WACANA",
		"CV Mitra Jaya":               "MITRA JAYA",
		"PT. Saripuri Permai Hotel":   "SARIPURI PERMAI HOTEL",
		"Astra Group":                 "ASTRA",
		"Bank Mandiri (Persero) Tbk.": "BANK MANDIRI",
	}
	for in, want := range cases {
		if got := NormalizeName(in); got != want {
			t.Errorf("NormalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHardKeyNPWPExact_BlocksRegardlessOfName(t *testing.T) {
	prospect := ProspectInput{RawCompanyName: "Completely Different Name", NPWP: "031234567801000"}
	candidates := []CandidateCompany{
		{ID: "c1", LegalName: "PT Cahaya Abadi Sejahtera", NormalizedName: "CAHAYA ABADI SEJAHTERA", NPWP: "031234567801000", Verified: true},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band != domain.BandBlock || out.Reason != "hard_key_npwp" {
		t.Fatalf("got band=%s reason=%s, want BLOCK/hard_key_npwp", out.Band, out.Reason)
	}
}

// Scenario 2: brand vs legal name.
func TestBrandAliasVerified_Blocks(t *testing.T) {
	prospect := ProspectInput{RawCompanyName: "Logisly"}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Logistik Canggih Indonesia",
			NormalizedName: NormalizeName("PT Logistik Canggih Indonesia"),
			BrandNames:     []string{NormalizeName("Logisly")},
			Verified:       true,
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band != domain.BandBlock || out.Reason != "brand_alias_match" {
		t.Fatalf("got band=%s reason=%s, want BLOCK/brand_alias_match", out.Band, out.Reason)
	}
}

// Same brand alias, but the company record itself is unverified (e.g. an
// AI-mock-suggested master entry) — FR4.3 says unverified data may never
// drive a hard BLOCK, so it must fall back to the soft-scoring path.
func TestBrandAliasUnverifiedCompany_NeverBlocksOnAliasAlone(t *testing.T) {
	prospect := ProspectInput{RawCompanyName: "Logisly"}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Logistik Canggih Indonesia",
			NormalizedName: NormalizeName("PT Logistik Canggih Indonesia"),
			BrandNames:     []string{NormalizeName("Logisly")},
			Verified:       false,
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band == domain.BandBlock {
		t.Fatalf("unverified brand alias must not BLOCK, got band=%s reason=%s", out.Band, out.Reason)
	}
}

// Scenario 3: group subsidiary, verified same_group edge — must land in
// REVIEW (corroborated by name+occupation similarity), never auto-BLOCK.
func TestGroupSubsidiaryVerified_Reviews(t *testing.T) {
	prospect := ProspectInput{
		RawCompanyName: "Bangun Sahabat Tani Indonesia", // realistic variant, not byte-identical to master
		OccupationLOB:  "agriculture",
	}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Bangun Sahabat Tani",
			NormalizedName:       NormalizeName("PT Bangun Sahabat Tani"),
			OccupationLOB:        "agriculture",
			Verified:             true,
			HasVerifiedGroupLink: true,
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band != domain.BandReview {
		t.Fatalf("got band=%s (score=%.2f), want REVIEW", out.Band, out.CompositeScore)
	}
}

// Same scenario but the group edge is only AI-suggested (unverified): even if
// the composite would otherwise cross the BLOCK threshold, it must be capped
// at REVIEW (FR4.3).
func TestGroupSubsidiaryUnverified_CappedAtReview(t *testing.T) {
	prospect := ProspectInput{
		RawCompanyName: "PT Bangun Sahabat Tani",
		Domain:         "dharmawibawaguna.co.id",
		Phone:          "0312345602",
		OccupationLOB:  "agriculture",
	}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Bangun Sahabat Tani",
			NormalizedName:         NormalizeName("PT Bangun Sahabat Tani"),
			Domain:                 "dharmawibawaguna.co.id",
			Phone:                  "0312345602",
			OccupationLOB:          "agriculture",
			Verified:               true,
			HasUnverifiedGroupLink: true,
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band == domain.BandBlock {
		t.Fatalf("unverified group link must never cross into BLOCK, got band=%s score=%.2f", out.Band, out.CompositeScore)
	}
	if out.Band != domain.BandReview {
		t.Fatalf("got band=%s, want REVIEW", out.Band)
	}
}

// Scenario 5: over-block guard. Two distinct companies sharing one office
// address must not exceed REVIEW (ideally PASS) — address+occupation alone
// are too weak to corroborate identity.
func TestOverBlockGuard_AddressAloneStaysAtOrBelowReview(t *testing.T) {
	prospect := ProspectInput{
		RawCompanyName: "PT Cipta Boga Nusantara",
		Address:        "Jl. Industri Raya No. 12, Surabaya",
		OccupationLOB:  "food_beverage",
	}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Maju Bersama Logistik",
			NormalizedName: NormalizeName("PT Maju Bersama Logistik"),
			Address:        "Jl. Industri Raya No. 12, Surabaya",
			OccupationLOB:  "logistics", // deliberately different LOB from the prospect
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band == domain.BandBlock {
		t.Fatalf("address-sharing alone must never BLOCK, got band=%s score=%.2f", out.Band, out.CompositeScore)
	}
	if out.Band != domain.BandPass {
		t.Fatalf("got band=%s score=%.2f, want PASS (distinct names/LOB, shared address only)", out.Band, out.CompositeScore)
	}
}

// FR3.3: address+occupation alone (even when both match) must stay <= REVIEW.
func TestOverBlockGuard_AddressAndOccupationNeverExceedsReview(t *testing.T) {
	prospect := ProspectInput{
		RawCompanyName: "Completely Unrelated Name",
		Address:        "Jl. Industri Raya No. 12, Surabaya",
		OccupationLOB:  "logistics",
	}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Maju Bersama Logistik",
			NormalizedName: NormalizeName("PT Maju Bersama Logistik"),
			Address:        "Jl. Industri Raya No. 12, Surabaya",
			OccupationLOB:  "logistics",
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band == domain.BandBlock {
		t.Fatalf("address+occupation alone must never BLOCK, got band=%s score=%.2f", out.Band, out.CompositeScore)
	}
}

// Fuzzy-name+address: a plausible near-duplicate name plus a shared address
// should corroborate into REVIEW (not PASS, not BLOCK).
func TestFuzzyNamePlusAddress_Reviews(t *testing.T) {
	prospect := ProspectInput{
		RawCompanyName: "PT Sinar Abadi Makmur Sejahtera",
		Address:        "Jl. Pahlawan No. 5, Surabaya",
		OccupationLOB:  "retail",
		Domain:         "sinarabadi.co.id",
	}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Sinar Abadi Makmur",
			NormalizedName: NormalizeName("PT Sinar Abadi Makmur"),
			Address:        "Jl. Pahlawan No. 5, Surabaya",
			OccupationLOB:  "retail",
			Domain:         "sinarabadi.co.id",
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.Band != domain.BandReview {
		t.Fatalf("got band=%s score=%.2f, want REVIEW", out.Band, out.CompositeScore)
	}
}

// The group bonus must not become a free floor score for every member of a
// verified group regardless of relevance — it should only corroborate a
// candidate that already has some baseline name/contact correlation.
func TestGroupBonus_DoesNotApplyWithoutBaselineCorrelation(t *testing.T) {
	prospect := ProspectInput{RawCompanyName: "PT Teknologi Inovasi Mandiri"}
	candidates := []CandidateCompany{
		{
			ID: "c1", LegalName: "PT Bangun Sahabat Tani",
			NormalizedName:       NormalizeName("PT Bangun Sahabat Tani"),
			Verified:             true,
			HasVerifiedGroupLink: true,
		},
	}

	out := Evaluate(prospect, candidates, defaultThresholds)
	if out.CompositeScore != 0 {
		t.Fatalf("got score=%.2f, want 0 (no name/contact correlation to justify the group bonus)", out.CompositeScore)
	}
	if out.Band != domain.BandPass {
		t.Fatalf("got band=%s, want PASS", out.Band)
	}
}

func TestEvaluate_NoCandidates_Passes(t *testing.T) {
	out := Evaluate(ProspectInput{RawCompanyName: "PT Belum Dikenal"}, nil, defaultThresholds)
	if out.Band != domain.BandPass {
		t.Fatalf("got band=%s, want PASS", out.Band)
	}
	if out.Candidates == nil {
		t.Fatalf("Candidates must be a non-nil empty slice for JSON encoding, got nil")
	}
}
