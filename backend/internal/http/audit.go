package http

import (
	"net/http"

	"ahid-prospect/backend/internal/domain"
)

// handleListAuditLogs implements FR8.1's acceptance: the audit role sees
// every state change in reverse-chronological order with actor + before/after.
func (s *Server) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireRole(r, domain.RoleAudit, domain.RoleAdmin); !ok {
		writeError(w, http.StatusForbidden, "only audit/admin may view the audit log")
		return
	}

	logs, err := s.Store.ListAuditLogs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}
	if logs == nil {
		logs = []domain.AuditLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}

func (s *Server) handleMonitoringSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.Store.ComputeMonitoringSummary(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to compute monitoring summary")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
