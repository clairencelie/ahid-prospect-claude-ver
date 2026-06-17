package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/domain"
	"ahid-prospect/backend/internal/store"
)

// handleRouteUW implements FR9.1: a PASS-band prospect proceeding to SPQ can
// be routed to UW, setting tanggal_masuk_uw to the server clock.
func (s *Server) handleRouteUW(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
	id := chi.URLParam(r, "id")

	latest, err := s.Store.GetLatestMatchResultForProspect(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusUnprocessableEntity, "prospect has no match decision yet")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load match decision")
		return
	}
	if latest.Band != domain.BandPass {
		writeError(w, http.StatusUnprocessableEntity, "only a PASS-band prospect may be routed to UW")
		return
	}

	prospect, err := s.Store.RouteToUW(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "prospect not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to route prospect to UW")
		return
	}

	if err := s.Store.WriteAudit(r.Context(), &user.ID, "prospect_routed_to_uw", "prospect", &id, nil, prospect); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusOK, prospect)
}

type closeUWRequest struct {
	Rationale string `json:"rationale"`
}

// handleCloseUW implements the rest of FR9.1/FR9.2: UW closes the item with
// a rationale; tanggal_keluar_uw is the server clock (consistent with the
// rest of the system never accepting a client-supplied timestamp, e.g.
// locks' locked_at) so the SLA duration it feeds can't be backdated.
func (s *Server) handleCloseUW(w http.ResponseWriter, r *http.Request) {
	user, ok := requireRole(r, domain.RoleUnderwriter)
	if !ok {
		writeError(w, http.StatusForbidden, "only underwriter may close a UW item")
		return
	}

	id := chi.URLParam(r, "id")

	var req closeUWRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Rationale == "" {
		writeError(w, http.StatusUnprocessableEntity, "rationale is required")
		return
	}

	before, err := s.Store.GetProspect(r.Context(), id)
	if err != nil {
		if err == store.ErrNotFound {
			writeError(w, http.StatusNotFound, "prospect not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load prospect")
		return
	}
	if before.TanggalMasukUW == nil {
		writeError(w, http.StatusConflict, "prospect was never routed to UW")
		return
	}

	after, err := s.Store.CloseUW(r.Context(), id, req.Rationale)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to close UW item")
		return
	}

	if err := s.Store.WriteAudit(r.Context(), &user.ID, "prospect_uw_closed", "prospect", &id, before, after); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusOK, after)
}
