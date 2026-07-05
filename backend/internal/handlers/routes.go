package handlers

import (
	"encoding/json"
	"net/http"

	"seo-auditor/backend/internal/database"
	"seo-auditor/backend/internal/reports"
)

type Dependencies struct {
	Reports   database.SeoReportRepository
	Generator reports.Generator
}

func RegisterRoutes(router *http.ServeMux, deps Dependencies) {
	handler := NewSeoReportHandler(deps.Reports, deps.Generator)

	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	router.HandleFunc("POST /seo-reports", handler.Create)
	router.HandleFunc("GET /seo-reports", handler.List)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
