package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/domain"
	"ahid-prospect/backend/internal/store"
)

type createRelationshipRequest struct {
	ParentCompanyID string  `json:"parent_company_id"`
	ChildCompanyID  string  `json:"child_company_id"`
	RelationType    string  `json:"relation_type"`
	Confidence      float64 `json:"confidence,omitempty"`
}

var validRelationTypes = map[string]bool{
	"subsidiary_of": true, "same_group": true, "brand_of": true, "dba": true,
}

func (s *Server) handleCreateRelationship(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(r, domain.RoleAdmin)
	if !ok {
		writeError(w, http.StatusForbidden, "only admin may manage the relationship graph")
		return
	}

	var req createRelationshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ParentCompanyID == "" || req.ChildCompanyID == "" || !validRelationTypes[req.RelationType] {
		writeError(w, http.StatusUnprocessableEntity, "parent_company_id, child_company_id and a valid relation_type are required")
		return
	}
	confidence := req.Confidence
	if confidence == 0 {
		confidence = 1.0
	}

	rel, err := s.Store.CreateRelationship(r.Context(), store.CreateRelationshipInput{
		ParentCompanyID: req.ParentCompanyID,
		ChildCompanyID:  req.ChildCompanyID,
		RelationType:    req.RelationType,
		Confidence:      confidence,
		Source:          "manual",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create relationship")
		return
	}

	if err := s.Store.WriteAudit(r.Context(), &user.ID, "relationship_created", "company_relationship", &rel.ID, nil, rel); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusCreated, rel)
}

type verifyRelationshipRequest struct {
	VerificationStatus domain.VerificationStatus `json:"verification_status"`
}

// handleVerifyRelationship implements FR4.4: admin can flip an edge's
// verification_status (verify an AI suggestion, or revert a verified edge
// back to unverified), always with an audit entry recording before/after.
func (s *Server) handleVerifyRelationship(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(r, domain.RoleAdmin)
	if !ok {
		writeError(w, http.StatusForbidden, "only admin may verify relationships")
		return
	}

	id := chi.URLParam(r, "id")

	var req verifyRelationshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.VerificationStatus != domain.Verified && req.VerificationStatus != domain.Unverified {
		writeError(w, http.StatusUnprocessableEntity, "verification_status must be 'verified' or 'unverified'")
		return
	}

	before, err := s.Store.GetRelationship(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "relationship not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load relationship")
		return
	}

	after, err := s.Store.SetRelationshipVerification(r.Context(), id, req.VerificationStatus)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update relationship")
		return
	}

	if err := s.Store.WriteAudit(r.Context(), &user.ID, "relationship_verified", "company_relationship", &id, before, after); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusOK, after)
}
