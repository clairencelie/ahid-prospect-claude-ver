package http

import (
	"context"
	"errors"
	"net/http"

	"ahid-prospect/backend/internal/domain"
	"ahid-prospect/backend/internal/store"
)

// corsMiddleware allows any origin, since this is a local demo where the
// frontend and API run on different ports (and different containers under
// docker-compose). Not suitable for production as-is.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Demo-Role, X-Demo-User-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type ctxKey string

const userCtxKey ctxKey = "demo_user"

// demoAuthMiddleware reads the X-Demo-Role / X-Demo-User-Id headers set by the
// frontend's role switcher (§10) and loads the matching seeded user. This is
// explicitly demo-only auth and must be replaced by real SSO/JWT in production.
func (s *Server) demoAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-Demo-User-Id")
		if userID == "" {
			writeError(w, http.StatusUnauthorized, "missing X-Demo-User-Id header")
			return
		}

		user, err := s.Store.GetUser(r.Context(), userID)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "unknown demo user")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load demo user")
			return
		}

		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userFromContext(ctx context.Context) *domain.User {
	u, _ := ctx.Value(userCtxKey).(*domain.User)
	return u
}

// requireRole returns 403 unless the current user's role is one of allowed.
func requireRole(r *http.Request, allowed ...domain.Role) (*domain.User, bool) {
	u := userFromContext(r.Context())
	if u == nil {
		return nil, false
	}
	for _, role := range allowed {
		if u.Role == role {
			return u, true
		}
	}
	return u, false
}
