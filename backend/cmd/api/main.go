package main

import (
	"context"
	"log"
	"net/http"

	httpserver "ahid-prospect/backend/internal/http"

	"ahid-prospect/backend/internal/config"
	"ahid-prospect/backend/internal/core"
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

	router := httpserver.NewRouter(server)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
