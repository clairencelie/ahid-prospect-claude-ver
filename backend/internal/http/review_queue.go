package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/domain"
	"ahid-prospect/backend/internal/store"
)

type reviewQueueResponse struct {
	MatchReviews  []domain.MatchResult  `json:"match_reviews"`
	LockConflicts []domain.ProspectLock `json:"lock_conflicts"`
}

// handleGetReviewQueue implements FR6.1: REVIEW-band match results still
// awaiting a checker decision, plus flagged lock conflicts (FR5.3/5.4).
func (s *Server) handleGetReviewQueue(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(r, domain.RoleBranchCo, domain.RoleCompliance, domain.RoleAudit, domain.RoleAdmin); !ok {
		writeError(w, http.StatusForbidden, "role not permitted to view the review queue")
		return
	}

	matchReviews, err := s.Store.ListReviewQueue(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list review queue")
		return
	}
	lockConflicts, err := s.Store.ListLockConflicts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list lock conflicts")
		return
	}
	if matchReviews == nil {
		matchReviews = []domain.MatchResult{}
	}
	if lockConflicts == nil {
		lockConflicts = []domain.ProspectLock{}
	}

	writeJSON(w, http.StatusOK, reviewQueueResponse{MatchReviews: matchReviews, LockConflicts: lockConflicts})
}

type makerDecisionRequest struct {
	Decision string `json:"decision"`
}

var validMakerDecisions = map[string]bool{
	"confirm_duplicate": true, "mark_distinct": true, "assign_owner": true,
}

// handleMakerDecision implements the maker half of FR6.2: a branch_co
// proposes a resolution for a REVIEW item.
func (s *Server) handleMakerDecision(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(r, domain.RoleBranchCo)
	if !ok {
		writeError(w, http.StatusForbidden, "only branch_co may make a review decision")
		return
	}

	id := chi.URLParam(r, "id")

	var req makerDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !validMakerDecisions[req.Decision] {
		writeError(w, http.StatusUnprocessableEntity, "decision must be one of confirm_duplicate, mark_distinct, assign_owner")
		return
	}

	before, err := s.Store.GetMatchResult(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "match result not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load match result")
		return
	}

	after, err := s.Store.SetMakerDecision(r.Context(), id, user.ID, req.Decision)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record maker decision")
		return
	}

	if err := s.Store.WriteAudit(r.Context(), &user.ID, "match_result_maker_decision", "match_result", &id, before, after); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusOK, after)
}

type checkerDecisionRequest struct {
	Approve bool `json:"approve"`
}

// handleCheckerDecision implements the checker half of FR6.2: a compliance
// user approves the maker's proposal (finalizing it) or rejects it (clearing
// the decision so the maker can reconsider). Always a distinct actor from
// the maker, and always audit-logged (FR6.2 acceptance).
func (s *Server) handleCheckerDecision(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(r, domain.RoleCompliance)
	if !ok {
		writeError(w, http.StatusForbidden, "only compliance may approve a review decision")
		return
	}

	id := chi.URLParam(r, "id")

	var req checkerDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	before, err := s.Store.GetMatchResult(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "match result not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load match result")
		return
	}
	if before.MakerID == nil {
		writeError(w, http.StatusConflict, "no maker decision is pending checker approval")
		return
	}

	after, err := s.Store.SetCheckerApproval(r.Context(), id, user.ID, req.Approve)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record checker decision")
		return
	}

	action := "match_result_checker_approved"
	if !req.Approve {
		action = "match_result_checker_rejected"
	}
	if err := s.Store.WriteAudit(r.Context(), &user.ID, action, "match_result", &id, before, after); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusOK, after)
}
