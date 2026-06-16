package http

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/domain"
	"ahid-prospect/backend/internal/matching"
	"ahid-prospect/backend/internal/store"
)

type checkRequest struct {
	RawCompanyName string  `json:"raw_company_name"`
	NPWP           *string `json:"npwp,omitempty"`
	NIB            *string `json:"nib,omitempty"`
	OccupationLOB  *string `json:"occupation_lob,omitempty"`
	Address        *string `json:"address,omitempty"`
	Domain         *string `json:"domain,omitempty"`
	Phone          *string `json:"phone,omitempty"`
	Headcount      *int    `json:"headcount,omitempty"`
}

type existingClientInfo struct {
	Found        bool    `json:"found"`
	PolicyNo     *string `json:"policy_no,omitempty"`
	Status       *string `json:"status,omitempty"`
	PeriodEnd    *string `json:"period_end,omitempty"`
	OwningBranch *string `json:"owning_branch,omitempty"`
}

type lockInfo struct {
	Protected bool    `json:"protected"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

type checkResult struct {
	Band           domain.Band        `json:"band"`
	Reason         string             `json:"reason"`
	CompositeScore float64            `json:"composite_score"`
	Candidates     []domain.Candidate `json:"candidates"`
	ExistingClient existingClientInfo `json:"existing_client"`
	Lock           lockInfo           `json:"lock"`
}

func (s *Server) handleProspectsCheck(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RawCompanyName == "" {
		writeError(w, http.StatusUnprocessableEntity, "raw_company_name is required")
		return
	}

	user := userFromContext(r.Context())
	result, err := s.evaluateProspect(r.Context(), user.Role, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to evaluate prospect")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// evaluateProspect runs the M2 existing-client check first (it short-circuits
// on an active policy), then the M3 matching engine against the company
// master. Owning-branch identity is masked for the `marketing` role.
func (s *Server) evaluateProspect(ctx context.Context, role domain.Role, req checkRequest) (*checkResult, error) {
	result := &checkResult{
		Band:           domain.BandPass,
		Reason:         "",
		CompositeScore: 0,
		Candidates:     []domain.Candidate{},
		ExistingClient: existingClientInfo{Found: false},
		Lock:           lockInfo{Protected: false},
	}

	if req.NPWP != nil && *req.NPWP != "" {
		npwp := matching.NormalizeDigits(*req.NPWP)
		policies, err := s.Core.ActivePoliciesByNPWP(ctx, npwp)
		if err != nil {
			return nil, err
		}
		if len(policies) > 0 {
			p := policies[0]
			result.Band = domain.BandBlock
			result.Reason = "existing_active_policy"
			result.ExistingClient = existingClientInfo{
				Found:     true,
				PolicyNo:  &p.PolicyNo,
				Status:    &p.Status,
				PeriodEnd: p.PeriodEnd,
			}
			if role != domain.RoleMarketing {
				result.ExistingClient.OwningBranch = p.OwningBranchID
			}
			return result, nil
		}
	}

	candidates, err := s.Store.ListCandidateCompanies(ctx)
	if err != nil {
		return nil, err
	}

	outcome := matching.Evaluate(toProspectInput(req), toMatchingCandidates(candidates), matching.Thresholds{
		BlockThreshold:  s.Config.BlockThreshold,
		ReviewThreshold: s.Config.ReviewThreshold,
	})
	result.Band = outcome.Band
	result.Reason = outcome.Reason
	result.CompositeScore = outcome.CompositeScore
	result.Candidates = topCandidates(outcome.Candidates, 5)

	return result, nil
}

func toProspectInput(req checkRequest) matching.ProspectInput {
	in := matching.ProspectInput{RawCompanyName: req.RawCompanyName}
	if req.NPWP != nil {
		in.NPWP = matching.NormalizeDigits(*req.NPWP)
	}
	if req.NIB != nil {
		in.NIB = matching.NormalizeDigits(*req.NIB)
	}
	if req.Domain != nil {
		in.Domain = *req.Domain
	}
	if req.Phone != nil {
		in.Phone = *req.Phone
	}
	if req.Address != nil {
		in.Address = *req.Address
	}
	if req.OccupationLOB != nil {
		in.OccupationLOB = *req.OccupationLOB
	}
	return in
}

func toMatchingCandidates(companies []store.CompanyCandidate) []matching.CandidateCompany {
	out := make([]matching.CandidateCompany, 0, len(companies))
	for _, cc := range companies {
		c := cc.Company
		mc := matching.CandidateCompany{
			ID:                     c.ID,
			LegalName:              c.LegalName,
			NormalizedName:         c.NormalizedName,
			BrandNames:             c.BrandNames,
			Verified:               c.VerificationStatus == domain.Verified,
			HasVerifiedGroupLink:   cc.HasVerifiedGroupLink,
			HasUnverifiedGroupLink: cc.HasUnverifiedGroupLink,
		}
		if c.NPWP != nil {
			mc.NPWP = *c.NPWP
		}
		if c.NIB != nil {
			mc.NIB = *c.NIB
		}
		if c.Domain != nil {
			mc.Domain = *c.Domain
		}
		if c.Phone != nil {
			mc.Phone = *c.Phone
		}
		if c.Address != nil {
			mc.Address = *c.Address
		}
		if c.OccupationLOB != nil {
			mc.OccupationLOB = *c.OccupationLOB
		}
		out = append(out, mc)
	}
	return out
}

// topCandidates keeps the response payload small: the strongest n candidates
// by score, since the master can grow far larger than what's useful to show.
func topCandidates(candidates []domain.Candidate, n int) []domain.Candidate {
	sorted := make([]domain.Candidate, len(candidates))
	copy(sorted, candidates)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Score > sorted[j].Score })
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	if sorted == nil {
		sorted = []domain.Candidate{}
	}
	return sorted
}

func (s *Server) handleCreateProspect(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RawCompanyName == "" {
		writeError(w, http.StatusUnprocessableEntity, "raw_company_name is required")
		return
	}

	user := userFromContext(r.Context())

	result, err := s.evaluateProspect(r.Context(), user.Role, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to evaluate prospect")
		return
	}

	prospect, err := s.Store.CreateProspect(r.Context(), store.CreateProspectInput{
		RawCompanyName: req.RawCompanyName,
		NPWP:           normalizedOrNil(req.NPWP),
		NIB:            normalizedOrNil(req.NIB),
		OccupationLOB:  req.OccupationLOB,
		Address:        req.Address,
		Domain:         req.Domain,
		Phone:          req.Phone,
		Headcount:      req.Headcount,
		BranchID:       user.BranchID,
		MarketingID:    &user.ID,
		Stage:          "indicative",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create prospect")
		return
	}

	var candidateID *string
	if len(result.Candidates) > 0 {
		candidateID = &result.Candidates[0].CompanyID
	}
	if _, err := s.Store.InsertMatchResult(r.Context(), store.InsertMatchResultInput{
		ProspectID:         prospect.ID,
		CandidateCompanyID: candidateID,
		CompositeScore:     result.CompositeScore,
		Band:               result.Band,
		Reason:             result.Reason,
		MatchedSignals:     candidateSignals(result.Candidates),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store match result")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"prospect": prospect,
		"check":    result,
	})
}

func candidateSignals(candidates []domain.Candidate) []domain.MatchedSignal {
	if len(candidates) == 0 {
		return []domain.MatchedSignal{}
	}
	return candidates[0].MatchedSignals
}

func normalizedOrNil(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	v := matching.NormalizeDigits(*s)
	return &v
}

func (s *Server) handleListProspects(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	var marketingID *string
	if user.Role == domain.RoleMarketing {
		marketingID = &user.ID
	}

	prospects, err := s.Store.ListProspects(r.Context(), marketingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list prospects")
		return
	}
	writeJSON(w, http.StatusOK, prospects)
}

func (s *Server) handleGetProspect(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	prospect, err := s.Store.GetProspect(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "prospect not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load prospect")
		return
	}
	writeJSON(w, http.StatusOK, prospect)
}
