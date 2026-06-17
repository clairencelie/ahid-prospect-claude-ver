package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"ahid-prospect/backend/internal/config"
	"ahid-prospect/backend/internal/core"
	"ahid-prospect/backend/internal/store"
)

type Server struct {
	Store  *store.Store
	Core   core.CoreClient
	Config *config.Config
}

func NewRouter(s *Server) http.Handler {
	r := chi.NewRouter()

	// Demo-only permissive CORS: the frontend (Vite dev server or its own
	// container) runs on a different origin/port than the API.
	r.Use(corsMiddleware)

	r.Get("/api/v1/healthz", s.handleHealthz)
	// Public (no demo-auth header required yet): lets the frontend's role
	// switcher list seeded users before any X-Demo-User-Id is set (§10).
	r.Get("/api/v1/users", s.handleListUsers)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.demoAuthMiddleware)

		r.Get("/me", s.handleMe)

		r.Post("/prospects/check", s.handleProspectsCheck)
		r.Post("/prospects", s.handleCreateProspect)
		r.Get("/prospects", s.handleListProspects)
		r.Get("/prospects/{id}", s.handleGetProspect)
		r.Post("/prospects/{id}/lock", s.handleLockProspect)
		r.Post("/prospects/{id}/route-uw", s.handleRouteUW)
		r.Post("/prospects/{id}/close-uw", s.handleCloseUW)

		r.Get("/companies", s.handleSearchCompanies)
		r.Post("/companies", s.handleCreateCompany)
		r.Get("/companies/{id}", s.handleGetCompany)
		r.Get("/companies/{id}/relationships", s.handleListCompanyRelationships)

		r.Post("/relationships", s.handleCreateRelationship)
		r.Patch("/relationships/{id}/verify", s.handleVerifyRelationship)

		r.Get("/review-queue", s.handleGetReviewQueue)
		r.Post("/match-results/{id}/maker", s.handleMakerDecision)
		r.Post("/match-results/{id}/checker", s.handleCheckerDecision)

		r.Post("/enrichment/jobs", s.handleEnqueueEnrichmentJob)
		r.Get("/enrichment/jobs/{id}", s.handleGetEnrichmentJob)

		r.Get("/audit-logs", s.handleListAuditLogs)
		r.Get("/monitoring/summary", s.handleMonitoringSummary)
	})

	return r
}
