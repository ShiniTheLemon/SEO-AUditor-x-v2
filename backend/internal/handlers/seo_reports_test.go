package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"seo-auditor/backend/internal/models"
	"seo-auditor/backend/internal/reports"
)

type fakeRepository struct {
	created *models.SeoReport
	list    []models.SeoReport
	err     error
}

func (repository *fakeRepository) Create(ctx context.Context, report *models.SeoReport) error {
	if repository.err != nil {
		return repository.err
	}

	repository.created = report
	return nil
}

func (repository *fakeRepository) ListRecent(ctx context.Context, limit int) ([]models.SeoReport, error) {
	if repository.err != nil {
		return nil, repository.err
	}

	return repository.list, nil
}

type fakeGenerator struct {
	report *reports.AuditReport
	raw    []byte
	err    error
}

func (generator fakeGenerator) Generate(ctx context.Context, targetURL string) (*reports.AuditReport, []byte, error) {
	if generator.err != nil {
		return nil, nil, generator.err
	}

	return generator.report, generator.raw, nil
}

func TestCreateSeoReport(t *testing.T) {
	raw := []byte(`{"title":"Audit","summary":"Summary","score":90}`)
	repository := &fakeRepository{}
	router := http.NewServeMux()
	RegisterRoutes(router, Dependencies{
		Reports: repository,
		Generator: fakeGenerator{
			report: &reports.AuditReport{
				Title:   "Audit",
				Summary: "Summary",
				Score:   90,
			},
			raw: raw,
		},
	})

	requestBody := bytes.NewBufferString(`{"url":"https://example.com"}`)
	request := httptest.NewRequest(http.MethodPost, "/seo-reports", requestBody)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d with body %s", http.StatusCreated, response.Code, response.Body.String())
	}

	if repository.created == nil {
		t.Fatal("expected repository Create to be called")
	}

	if repository.created.URL != "https://example.com" {
		t.Fatalf("expected stored URL, got %q", repository.created.URL)
	}

	if string(repository.created.Report) != string(raw) {
		t.Fatalf("expected stored raw report %s, got %s", raw, repository.created.Report)
	}
}

func TestCreateSeoReportReturnsBadRequestForGeneratorError(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Dependencies{
		Reports: &fakeRepository{},
		Generator: fakeGenerator{
			err: errors.New("a valid URL is required"),
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/seo-reports", bytes.NewBufferString(`{"url":"bad"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestListSeoReports(t *testing.T) {
	router := http.NewServeMux()
	RegisterRoutes(router, Dependencies{
		Reports: &fakeRepository{
			list: []models.SeoReport{
				{URL: "https://example.com", Title: "Audit", Score: 90},
			},
		},
		Generator: fakeGenerator{},
	})

	request := httptest.NewRequest(http.MethodGet, "/seo-reports", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var payload []models.SeoReport
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}

	if len(payload) != 1 {
		t.Fatalf("expected 1 report, got %d", len(payload))
	}
}
