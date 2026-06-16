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

	r.Get("/api/v1/healthz", s.handleHealthz)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.demoAuthMiddleware)

		r.Get("/me", s.handleMe)

		r.Post("/prospects/check", s.handleProspectsCheck)
		r.Post("/prospects", s.handleCreateProspect)
		r.Get("/prospects", s.handleListProspects)
		r.Get("/prospects/{id}", s.handleGetProspect)

		r.Get("/companies", s.handleSearchCompanies)
		r.Post("/companies", s.handleCreateCompany)
		r.Get("/companies/{id}", s.handleGetCompany)
		r.Get("/companies/{id}/relationships", s.handleListCompanyRelationships)

		r.Post("/relationships", s.handleCreateRelationship)
		r.Patch("/relationships/{id}/verify", s.handleVerifyRelationship)
	})

	return r
}
