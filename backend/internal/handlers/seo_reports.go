package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"seo-auditor/backend/internal/database"
	"seo-auditor/backend/internal/identity"
	"seo-auditor/backend/internal/models"
	"seo-auditor/backend/internal/reports"
)

type SeoReportHandler struct {
	reports   database.SeoReportRepository
	generator reports.Generator
}

type createSeoReportRequest struct {
	URL               string          `json:"url"`
	DeviceFingerprint string          `json:"device_fingerprint"`
	DeviceDetails     json.RawMessage `json:"device_details"`
}

func NewSeoReportHandler(repository database.SeoReportRepository, generator reports.Generator) *SeoReportHandler {
	return &SeoReportHandler{
		reports:   repository,
		generator: generator,
	}
}

func (handler *SeoReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createSeoReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A valid JSON body is required."})
		return
	}

	userID := identity.UserIDFromFingerprint(request.DeviceFingerprint)
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Device fingerprint is required."})
		return
	}

	deviceDetails := request.DeviceDetails
	if len(deviceDetails) == 0 {
		deviceDetails = json.RawMessage(`{}`)
	}

	log.Printf("create report started url=%q", request.URL)
	generatedReport, rawReport, err := handler.generator.Generate(r.Context(), request.URL)
	if err != nil {
		log.Printf("create report generation failed url=%q error=%v", request.URL, err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	log.Printf("create report generated url=%q title=%q score=%d checks=%d recommendations=%d", request.URL, generatedReport.Title, generatedReport.Score, len(generatedReport.Checks), len(generatedReport.Recommendations))

	record := models.SeoReport{
		URL:           request.URL,
		Title:         generatedReport.Title,
		Summary:       generatedReport.Summary,
		Score:         generatedReport.Score,
		Report:        rawReport,
		UserID:        userID,
		DeviceDetails: deviceDetails,
	}

	if err := handler.reports.Create(r.Context(), &record); err != nil {
		log.Printf("failed to save SEO report: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save SEO report.", "details": err.Error()})
		return
	}

	log.Printf("create report saved id=%s url=%q", record.ID, record.URL)
	writeJSON(w, http.StatusCreated, record)
}

func (handler *SeoReportHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := identity.UserIDFromFingerprint(r.URL.Query().Get("device_fingerprint"))
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Device fingerprint is required."})
		return
	}

	records, err := handler.reports.ListRecentByUser(r.Context(), userID, 10)
	if err != nil {
		log.Printf("failed to load SEO reports: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to load SEO reports.", "details": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, records)
}
