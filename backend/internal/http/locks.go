package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/domain"
	"ahid-prospect/backend/internal/store"
)

// lockKey is the dedup key locks are granted against. The prospect_locks
// schema only has a single `npwp` text column (FR5.1 frames locks as
// NPWP-based), so when only a NIB was captured at the protection gate we key
// the lock on that NIB value instead — it's still the unique identifier the
// protection-gate required (FR1.2).
func lockKey(p *domain.Prospect) string {
	if p.NPWP != nil && *p.NPWP != "" {
		return *p.NPWP
	}
	if p.NIB != nil && *p.NIB != "" {
		return *p.NIB
	}
	return ""
}

// handleLockProspect implements M5: first-come protection lock with a
// server-clock timestamp, time-boxed TTL, and need-to-know conflict
// handling (FR5.1-5.4) — a competing branch is told the prospect is
// protected without ever learning who holds it.
func (s *Server) handleLockProspect(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())
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

	key := lockKey(prospect)
	if key == "" {
		writeError(w, http.StatusUnprocessableEntity, "npwp or nib is required to request a protection lock")
		return
	}

	released, err := s.Store.ReleaseExpiredLock(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to release expired locks")
		return
	}
	if released != nil {
		if err := s.Store.WriteAudit(r.Context(), nil, "lock_expired_released", "prospect_lock", &released.ID, nil, released); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write audit log")
			return
		}
	}

	existing, err := s.Store.GetActiveLock(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check existing lock")
		return
	}

	if existing != nil {
		if existing.ProspectID == prospect.ID {
			// Idempotent: the same prospect re-requesting its own active lock.
			writeJSON(w, http.StatusOK, lockInfo{Protected: true, ExpiresAt: timePtr(existing.ExpiresAt)})
			return
		}

		// Need-to-know (FR5.3): never reveal the holding branch/marketer to
		// the competing requester. Record the conflicting claim for compliance.
		conflict, err := s.Store.CreateLock(r.Context(), store.CreateLockInput{
			ProspectID:  prospect.ID,
			CompanyID:   prospect.CompanyID,
			NPWP:        key,
			BranchID:    user.BranchID,
			MarketingID: &user.ID,
			TTLDays:     s.Config.LockTTLDays,
			Status:      "in_conflict",
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to record lock conflict")
			return
		}
		if err := s.Store.WriteAudit(r.Context(), &user.ID, "lock_conflict_flagged", "prospect_lock", &conflict.ID, nil, conflict); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write audit log")
			return
		}

		writeJSON(w, http.StatusConflict, lockInfo{Protected: true})
		return
	}

	lock, err := s.Store.CreateLock(r.Context(), store.CreateLockInput{
		ProspectID:  prospect.ID,
		CompanyID:   prospect.CompanyID,
		NPWP:        key,
		BranchID:    user.BranchID,
		MarketingID: &user.ID,
		TTLDays:     s.Config.LockTTLDays,
		Status:      "active",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create lock")
		return
	}

	if err := s.Store.SetProspectStage(r.Context(), prospect.ID, "protection"); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update prospect stage")
		return
	}
	if err := s.Store.WriteAudit(r.Context(), &user.ID, "lock_granted", "prospect_lock", &lock.ID, nil, lock); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write audit log")
		return
	}

	writeJSON(w, http.StatusCreated, lockInfo{Protected: true, ExpiresAt: timePtr(lock.ExpiresAt)})
}

func timePtr(t time.Time) *string {
	s := t.Format(time.RFC3339)
	return &s
}
