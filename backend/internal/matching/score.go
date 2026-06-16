package matching

import (
	"strings"

	"ahid-prospect/backend/internal/domain"
)

// Weights from FR3.1. NPWP/NIB are decisive (handled as a short-circuit, not
// a weighted contribution) so they aren't listed here.
const (
	maxNameContribution       = 35
	maxGroupContribution      = 30
	maxDomainContribution     = 15
	maxPhoneContribution      = 10
	maxAddressContribution    = 10
	maxOccupationContribution = 5

	brandAliasThreshold = 0.92
)

type Thresholds struct {
	BlockThreshold  float64
	ReviewThreshold float64
}

// ProspectInput is the already-collected (but not yet normalized) prospect
// data being checked against the company master.
type ProspectInput struct {
	RawCompanyName string
	NPWP           string // normalized digits, "" if absent
	NIB            string
	Domain         string
	Phone          string
	Address        string
	OccupationLOB  string
}

// CandidateCompany is a row from the company master plus the relationship-
// graph facts the store has already resolved (so this package never touches
// the DB directly and stays unit-testable).
type CandidateCompany struct {
	ID                     string
	LegalName              string
	NormalizedName         string
	BrandNames             []string // normalized
	NPWP                   string
	NIB                    string
	Domain                 string
	Phone                  string
	Address                string
	OccupationLOB          string
	Verified               bool // company record's own verification_status
	HasVerifiedGroupLink   bool // verified same_group/subsidiary_of/brand_of/dba edge
	HasUnverifiedGroupLink bool // only unverified (ai_mock) edges
}

type Outcome struct {
	Band           domain.Band
	Reason         string
	CompositeScore float64
	Candidates     []domain.Candidate
}

// Evaluate scores the prospect against every candidate (FR3.1), assigns each
// a band (FR3.2/FR3.3), and returns the strongest result as the top-level
// decision alongside the full ranked candidate list for explainability (FR3.4).
func Evaluate(prospect ProspectInput, candidates []CandidateCompany, th Thresholds) Outcome {
	pname := NormalizeName(prospect.RawCompanyName)

	var scored []domain.Candidate
	var bestBand = domain.BandPass
	var bestReason string
	var bestScore float64
	bestRank := -1

	for _, c := range candidates {
		band, reason, composite, signals := evaluateCandidate(pname, prospect, c, th)
		scored = append(scored, domain.Candidate{
			CompanyID:      c.ID,
			LegalName:      c.LegalName,
			Score:          composite,
			MatchedSignals: signals,
		})

		rank := bandRank(band)
		if rank > bestRank || (rank == bestRank && composite > bestScore) {
			bestRank = rank
			bestBand = band
			bestReason = reason
			bestScore = composite
		}
	}

	if scored == nil {
		scored = []domain.Candidate{}
	}

	return Outcome{
		Band:           bestBand,
		Reason:         bestReason,
		CompositeScore: bestScore,
		Candidates:     scored,
	}
}

func bandRank(b domain.Band) int {
	switch b {
	case domain.BandBlock:
		return 2
	case domain.BandReview:
		return 1
	default:
		return 0
	}
}

func evaluateCandidate(pname string, p ProspectInput, c CandidateCompany, th Thresholds) (domain.Band, string, float64, []domain.MatchedSignal) {
	// Hard keys are decisive (FR3.1): exact NPWP/NIB means same entity.
	if p.NPWP != "" && c.NPWP != "" && p.NPWP == c.NPWP {
		return domain.BandBlock, "hard_key_npwp", 100, []domain.MatchedSignal{
			{Signal: "npwp_exact", Similarity: 1, Contribution: 100},
		}
	}
	if p.NIB != "" && c.NIB != "" && p.NIB == c.NIB {
		return domain.BandBlock, "hard_key_nib", 100, []domain.MatchedSignal{
			{Signal: "nib_exact", Similarity: 1, Contribution: 100},
		}
	}

	// A verified brand alias on the golden record is as trustworthy as a hard
	// key (it's curated master data, not a fuzzy guess) — but only verified;
	// an AI-suggested alias must go through the soft scoring path (FR4.3).
	if c.Verified {
		for _, brand := range c.BrandNames {
			if NameSimilarity(pname, brand) >= brandAliasThreshold {
				return domain.BandBlock, "brand_alias_match", 100, []domain.MatchedSignal{
					{Signal: "brand_alias", Similarity: NameSimilarity(pname, brand), Contribution: 100},
				}
			}
		}
	}

	var signals []domain.MatchedSignal
	var composite float64

	nameSim := NameSimilarity(pname, c.NormalizedName)
	for _, brand := range c.BrandNames {
		if s := NameSimilarity(pname, brand); s > nameSim {
			nameSim = s
		}
	}
	nameContribution := nameSim * maxNameContribution
	signals = append(signals, domain.MatchedSignal{Signal: "name_similarity", Similarity: round2(nameSim), Contribution: round2(nameContribution)})
	composite += nameContribution

	domainMatched := p.Domain != "" && c.Domain != "" && strings.EqualFold(p.Domain, c.Domain)
	phoneMatched := p.Phone != "" && c.Phone != "" && NormalizeDigits(p.Phone) == NormalizeDigits(c.Phone)

	// The group bonus only corroborates an *already plausible* identity match
	// (by name or exact contact info) — without this gate, every member of a
	// verified group would get a free 30-point floor against any unrelated
	// prospect just for being in the master, which isn't what FR3.1 intends.
	groupVerified := true
	if (c.HasVerifiedGroupLink || c.HasUnverifiedGroupLink) && (nameSim >= 0.3 || domainMatched || phoneMatched) {
		groupVerified = c.HasVerifiedGroupLink
		signalName := "same_group"
		if !groupVerified {
			signalName = "same_group_unverified"
		}
		signals = append(signals, domain.MatchedSignal{Signal: signalName, Similarity: 1, Contribution: maxGroupContribution})
		composite += maxGroupContribution
	}

	if domainMatched {
		signals = append(signals, domain.MatchedSignal{Signal: "domain_match", Similarity: 1, Contribution: maxDomainContribution})
		composite += maxDomainContribution
	}

	if phoneMatched {
		signals = append(signals, domain.MatchedSignal{Signal: "phone_match", Similarity: 1, Contribution: maxPhoneContribution})
		composite += maxPhoneContribution
	}

	if p.Address != "" && c.Address != "" {
		addrSim := NameSimilarity(NormalizeText(p.Address), NormalizeText(c.Address))
		addrContribution := addrSim * maxAddressContribution
		signals = append(signals, domain.MatchedSignal{Signal: "address_similarity", Similarity: round2(addrSim), Contribution: round2(addrContribution)})
		composite += addrContribution
	}

	if p.OccupationLOB != "" && c.OccupationLOB != "" && strings.EqualFold(p.OccupationLOB, c.OccupationLOB) {
		signals = append(signals, domain.MatchedSignal{Signal: "occupation_match", Similarity: 1, Contribution: maxOccupationContribution})
		composite += maxOccupationContribution
	}

	if composite > 100 {
		composite = 100
	}

	band := domain.BandPass
	reason := ""
	switch {
	case composite >= th.BlockThreshold:
		if !groupVerified {
			// FR4.3: an unverified (AI-suggested) edge can push into REVIEW
			// but must never be the thing that crosses into BLOCK.
			band, reason = domain.BandReview, "same_group_unverified"
		} else {
			band, reason = domain.BandBlock, "composite_score"
		}
	case composite >= th.ReviewThreshold:
		band, reason = domain.BandReview, "composite_score"
	}

	if signals == nil {
		signals = []domain.MatchedSignal{}
	}
	return band, reason, round2(composite), signals
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
