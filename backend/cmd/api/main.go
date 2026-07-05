package main

import (
	"log"
	"net/http"

	"seo-auditor/backend/internal/config"
	"seo-auditor/backend/internal/database"
	"seo-auditor/backend/internal/handlers"
	"seo-auditor/backend/internal/reports"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.EnsureSchema(db); err != nil {
		log.Fatal(err)
	}

	router := http.NewServeMux()
	repository := database.NewSQLSeoReportRepository(db)
	generator := reports.NewGenerator(reports.GeneratorConfig{
		HTTPClient:        http.DefaultClient,
		OpenAIAPIKey:      cfg.OpenAIAPIKey,
		OpenAIModel:       cfg.OpenAIModel,
		OpenAIBaseURL:     cfg.OpenAIBaseURL,
		PageSpeedAPIKey:   cfg.PageSpeedKey,
		PageSpeedStrategy: cfg.PageSpeedMode,
	})

	handlers.RegisterRoutes(router, handlers.Dependencies{
		Reports:   repository,
		Generator: generator,
	})

	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, handlers.WithCORS(router)); err != nil {
		log.Fatal(err)
	}
}
