package main

import (
	"context"
	"log"
	"net/http"
	"time"

	httpserver "ahid-prospect/backend/internal/http"

	"ahid-prospect/backend/internal/config"
	"ahid-prospect/backend/internal/core"
	"ahid-prospect/backend/internal/enrichment"
	"ahid-prospect/backend/internal/store"
)

func main() {
	ctx := context.Background()

	cfg := config.FromEnv()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer st.Close()

	if err := cfg.LoadSettings(ctx, st.Pool); err != nil {
		log.Fatalf("failed to load settings: %v", err)
	}

	coreClient := core.NewMockCoreClient(st.Pool)

	server := &httpserver.Server{
		Store:  st,
		Core:   coreClient,
		Config: &cfg,
	}

	fixtures, err := enrichment.LoadFixtures(cfg.FixturesPath)
	if err != nil {
		log.Fatalf("failed to load enrichment fixtures: %v", err)
	}

	var gemini *enrichment.GeminiClient
	if cfg.GeminiAPIKey != "" {
		gemini = enrichment.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel)
		log.Printf("enrichment: gemini enabled (model=%s), fixtures used as fallback", gemini.Model)
	} else {
		log.Printf("enrichment: GEMINI_API_KEY not set, using fixtures only")
	}

	worker := &enrichment.Worker{
		Store:    st,
		Fixtures: fixtures,
		Gemini:   gemini,
		Interval: 3 * time.Second,
	}
	go worker.Start(ctx)

	router := httpserver.NewRouter(server)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
