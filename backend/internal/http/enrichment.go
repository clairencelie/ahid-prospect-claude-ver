package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/matching"
	"ahid-prospect/backend/internal/store"
)

type enqueueEnrichmentRequest struct {
	CompanyName string `json:"company_name"`
}

func (s *Server) handleEnqueueEnrichmentJob(w http.ResponseWriter, r *http.Request) {
	var req enqueueEnrichmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.CompanyName == "" {
		writeError(w, http.StatusUnprocessableEntity, "company_name is required")
		return
	}

	job, err := s.Store.EnqueueEnrichmentJob(r.Context(), req.CompanyName, matching.NormalizeName(req.CompanyName))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue enrichment job")
		return
	}

	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) handleGetEnrichmentJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := s.Store.GetEnrichmentJob(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "enrichment job not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load enrichment job")
		return
	}
	writeJSON(w, http.StatusOK, job)
}
