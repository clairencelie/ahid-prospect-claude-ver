package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/domain"
	"ahid-prospect/backend/internal/matching"
	"ahid-prospect/backend/internal/store"
)

func (s *Server) handleSearchCompanies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	companies, err := s.Store.SearchCompanies(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search companies")
		return
	}
	writeJSON(w, http.StatusOK, companies)
}

func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	company, err := s.Store.GetCompany(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "company not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load company")
		return
	}
	writeJSON(w, http.StatusOK, company)
}

type createCompanyRequest struct {
	NPWP          *string  `json:"npwp,omitempty"`
	NIB           *string  `json:"nib,omitempty"`
	LegalName     string   `json:"legal_name"`
	BrandNames    []string `json:"brand_names,omitempty"`
	Domain        *string  `json:"domain,omitempty"`
	Phone         *string  `json:"phone,omitempty"`
	Address       *string  `json:"address,omitempty"`
	OccupationLOB *string  `json:"occupation_lob,omitempty"`
	GroupID       *string  `json:"group_id,omitempty"`
}

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(r, domain.RoleAdmin)
	if !ok {
		writeError(w, http.StatusForbidden, "only admin may manage the company master")
		return
	}

	var req createCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.LegalName == "" {
		writeError(w, http.StatusUnprocessableEntity, "legal_name is required")
		return
	}

	brandNames := make([]string, 0, len(req.BrandNames))
	for _, b := range req.BrandNames {
		brandNames = append(brandNames, matching.NormalizeName(b))
	}

	company, err := s.Store.CreateCompany(r.Context(), store.CreateCompanyInput{
		NPWP:          normalizedOrNil(req.NPWP),
		NIB:           normalizedOrNil(req.NIB),
		LegalName:     req.LegalName,
		BrandNames:    brandNames,
		Domain:        req.Domain,
		Phone:         req.Phone,
		Address:       req.Address,
		OccupationLOB: req.OccupationLOB,
		GroupID:       req.GroupID,
		Source:        "manual",
	}, matching.NormalizeName(req.LegalName))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create company")
		return
	}

	if err := s.Store.WriteAudit(r.Context(), &user.ID, "company_created", "company", &company.ID, nil, company); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusCreated, company)
}

func (s *Server) handleListCompanyRelationships(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	relationships, err := s.Store.ListRelationshipsForCompany(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list relationships")
		return
	}
	writeJSON(w, http.StatusOK, relationships)
}
