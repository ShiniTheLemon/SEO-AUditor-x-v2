package reports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

const pageSpeedEndpoint = "https://pagespeedonline.googleapis.com/pagespeedonline/v5/runPagespeed"
const defaultOpenAIBaseURL = "https://jp.gpt.ge/v1"

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Generator interface {
	Generate(ctx context.Context, targetURL string) (*AuditReport, []byte, error)
}

type GeneratorConfig struct {
	HTTPClient        HTTPDoer
	OpenAIAPIKey      string
	OpenAIModel       string
	OpenAIBaseURL     string
	PageSpeedAPIKey   string
	PageSpeedStrategy string
}

type Service struct {
	httpClient        HTTPDoer
	openAIAPIKey      string
	openAIModel       string
	openAIBaseURL     string
	pageSpeedAPIKey   string
	pageSpeedStrategy string
}

type AuditReport struct {
	Title           string        `json:"title"`
	Summary         string        `json:"summary"`
	Score           int           `json:"score"`
	Checks          []ReportCheck `json:"checks"`
	Recommendations []string      `json:"recommendations"`
	PageSpeed       PageSnapshot  `json:"pagespeed"`
}

type ReportCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

type PageSnapshot struct {
	RequestedURL  string          `json:"requested_url"`
	FinalURL      string          `json:"final_url"`
	Strategy      string          `json:"strategy"`
	FetchedAt     string          `json:"fetched_at"`
	Categories    []CategoryScore `json:"categories"`
	Opportunities []AuditFinding  `json:"opportunities"`
	Diagnostics   []AuditFinding  `json:"diagnostics"`
	Warnings      []string        `json:"warnings"`
}

type CategoryScore struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Score int    `json:"score"`
}

type AuditFinding struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Score       *int   `json:"score,omitempty"`
	Display     string `json:"display,omitempty"`
}

type pageSpeedResponse struct {
	ID                   string `json:"id"`
	AnalysisUTCTimestamp string `json:"analysisUTCTimestamp"`
	LighthouseResult     struct {
		RequestedURL string `json:"requestedUrl"`
		FinalURL     string `json:"finalUrl"`
		RunWarnings  []any  `json:"runWarnings"`
		RuntimeError *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"runtimeError"`
		Categories map[string]struct {
			ID    string   `json:"id"`
			Title string   `json:"title"`
			Score *float64 `json:"score"`
		} `json:"categories"`
		Audits map[string]struct {
			ID               string   `json:"id"`
			Title            string   `json:"title"`
			Description      string   `json:"description"`
			Score            *float64 `json:"score"`
			ScoreDisplayMode string   `json:"scoreDisplayMode"`
			DisplayValue     string   `json:"displayValue"`
		} `json:"audits"`
	} `json:"lighthouseResult"`
}

func NewGenerator(config GeneratorConfig) *Service {
	client := config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}

	model := config.OpenAIModel
	if model == "" {
		model = "gpt-4.1-mini"
	}

	baseURL := strings.TrimSpace(config.OpenAIBaseURL)
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	strategy := strings.ToLower(config.PageSpeedStrategy)
	if strategy != "desktop" {
		strategy = "mobile"
	}

	return &Service{
		httpClient:        client,
		openAIAPIKey:      config.OpenAIAPIKey,
		openAIModel:       model,
		openAIBaseURL:     baseURL,
		pageSpeedAPIKey:   config.PageSpeedAPIKey,
		pageSpeedStrategy: strategy,
	}
}

func (service *Service) Generate(ctx context.Context, targetURL string) (*AuditReport, []byte, error) {
	normalizedURL, err := normalizeURL(targetURL)
	if err != nil {
		return nil, nil, err
	}

	if service.openAIAPIKey == "" {
		return nil, nil, errors.New("OPENAI_API_KEY is required")
	}

	log.Printf("report workflow: pagespeed request starting url=%q strategy=%s", normalizedURL, service.pageSpeedStrategy)
	pageSpeed, err := service.fetchPageSpeed(ctx, normalizedURL)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("report workflow: pagespeed request completed url=%q final_url=%q categories=%d audits=%d", normalizedURL, pageSpeed.LighthouseResult.FinalURL, len(pageSpeed.LighthouseResult.Categories), len(pageSpeed.LighthouseResult.Audits))

	snapshot := summarizePageSpeed(pageSpeed, service.pageSpeedStrategy)
	log.Printf("report workflow: pagespeed summary prepared url=%q opportunities=%d diagnostics=%d warnings=%d", normalizedURL, len(snapshot.Opportunities), len(snapshot.Diagnostics), len(snapshot.Warnings))

	log.Printf("report workflow: openai request starting url=%q model=%s base_url=%s", normalizedURL, service.openAIModel, service.openAIBaseURL)
	report, err := service.generateOpenAIReport(ctx, snapshot)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("report workflow: openai request completed url=%q title=%q score=%d", normalizedURL, report.Title, report.Score)
	report.PageSpeed = snapshot

	raw, err := json.Marshal(report)
	if err != nil {
		return nil, nil, err
	}

	return report, raw, nil
}

func normalizeURL(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("a valid URL is required")
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return "", errors.New("a valid URL is required")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("only http and https URLs are supported")
	}

	if parsed.Host == "" {
		return "", errors.New("a valid URL is required")
	}

	return parsed.String(), nil
}

func (service *Service) fetchPageSpeed(ctx context.Context, targetURL string) (*pageSpeedResponse, error) {
	requestURL, err := url.Parse(pageSpeedEndpoint)
	if err != nil {
		return nil, err
	}

	query := requestURL.Query()
	query.Set("url", targetURL)
	query.Set("strategy", strings.ToUpper(service.pageSpeedStrategy))
	query.Set("locale", "en")
	for _, category := range []string{"PERFORMANCE", "SEO", "ACCESSIBILITY", "BEST_PRACTICES"} {
		query.Add("category", category)
	}
	if service.pageSpeedAPIKey != "" {
		query.Set("key", service.pageSpeedAPIKey)
	}
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := service.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("pagespeed request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var payload pageSpeedResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	if payload.LighthouseResult.RuntimeError != nil {
		return nil, fmt.Errorf("pagespeed runtime error: %s", payload.LighthouseResult.RuntimeError.Message)
	}

	return &payload, nil
}

func summarizePageSpeed(payload *pageSpeedResponse, strategy string) PageSnapshot {
	snapshot := PageSnapshot{
		RequestedURL:  payload.LighthouseResult.RequestedURL,
		FinalURL:      payload.LighthouseResult.FinalURL,
		Strategy:      strategy,
		FetchedAt:     payload.AnalysisUTCTimestamp,
		Categories:    make([]CategoryScore, 0, len(payload.LighthouseResult.Categories)),
		Opportunities: make([]AuditFinding, 0),
		Diagnostics:   make([]AuditFinding, 0),
		Warnings:      stringifyWarnings(payload.LighthouseResult.RunWarnings),
	}

	if snapshot.FinalURL == "" {
		snapshot.FinalURL = payload.ID
	}

	for _, categoryID := range []string{"performance", "seo", "accessibility", "best-practices"} {
		category, ok := payload.LighthouseResult.Categories[categoryID]
		if !ok {
			continue
		}

		score := 0
		if category.Score != nil {
			score = int(*category.Score * 100)
		}

		snapshot.Categories = append(snapshot.Categories, CategoryScore{
			ID:    categoryID,
			Title: category.Title,
			Score: score,
		})
	}

	for auditID, audit := range payload.LighthouseResult.Audits {
		if audit.Score == nil {
			continue
		}

		score := int(*audit.Score * 100)
		if score >= 90 {
			continue
		}

		finding := AuditFinding{
			ID:          auditID,
			Title:       audit.Title,
			Description: cleanDescription(audit.Description),
			Score:       &score,
			Display:     audit.DisplayValue,
		}

		if audit.ScoreDisplayMode == "numeric" || audit.DisplayValue != "" {
			snapshot.Opportunities = append(snapshot.Opportunities, finding)
			continue
		}

		snapshot.Diagnostics = append(snapshot.Diagnostics, finding)
	}

	limitFindings(&snapshot.Opportunities, 8)
	limitFindings(&snapshot.Diagnostics, 8)

	return snapshot
}

func stringifyWarnings(warnings []any) []string {
	output := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		switch value := warning.(type) {
		case string:
			if value != "" {
				output = append(output, value)
			}
		default:
			raw, err := json.Marshal(value)
			if err == nil && string(raw) != "null" {
				output = append(output, string(raw))
			}
		}
	}

	return output
}

func cleanDescription(value string) string {
	value = strings.ReplaceAll(value, "[", "")
	value = strings.ReplaceAll(value, "]", "")
	value = strings.ReplaceAll(value, "`", "")
	return strings.TrimSpace(value)
}

func limitFindings(findings *[]AuditFinding, limit int) {
	if len(*findings) > limit {
		*findings = (*findings)[:limit]
	}
}

func (service *Service) generateOpenAIReport(ctx context.Context, snapshot PageSnapshot) (*AuditReport, error) {
	snapshotJSON, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, err
	}

	input := fmt.Sprintf(`Create a concise SEO audit report from this PageSpeed/Lighthouse data.

Rules:
- Score must be an integer from 0 to 100.
- Checks must use status values: pass, warning, or fail.
- Focus on SEO, performance, accessibility, and best-practice issues that affect search visibility.
- Recommendations must be concrete engineering actions.

PageSpeed summary:
%s

Return JSON shaped exactly like:
{
  "title": "Short title",
  "summary": "Brief summary",
  "score": 0,
  "checks": [
    {"name": "Check name", "status": "pass", "details": "What was found"}
  ],
  "recommendations": ["Action 1"]
}`, string(snapshotJSON))

	client := openai.NewClient(
		option.WithAPIKey(service.openAIAPIKey),
		option.WithBaseURL(service.openAIBaseURL),
	)

	response, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Instructions: openai.String("You are a technical SEO auditor. Return only valid JSON with title, summary, score, checks, and recommendations."),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(input),
		},
		Model: shared.ResponsesModel(service.openAIModel),
		Text: responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name:   "seo_audit_report",
					Schema: auditReportJSONSchema(),
					Strict: openai.Bool(true),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}

	text := response.OutputText()
	if text == "" {
		return nil, errors.New("openai response did not include output text")
	}

	var report AuditReport
	if err := json.Unmarshal([]byte(text), &report); err != nil {
		return nil, err
	}

	if report.Score < 0 {
		report.Score = 0
	}
	if report.Score > 100 {
		report.Score = 100
	}

	return &report, nil
}

func auditReportJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"title", "summary", "score", "checks", "recommendations"},
		"properties": map[string]any{
			"title": map[string]any{
				"type": "string",
			},
			"summary": map[string]any{
				"type": "string",
			},
			"score": map[string]any{
				"type":    "integer",
				"minimum": 0,
				"maximum": 100,
			},
			"checks": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"name", "status", "details"},
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
						"status": map[string]any{
							"type": "string",
							"enum": []string{"pass", "warning", "fail"},
						},
						"details": map[string]any{
							"type": "string",
						},
					},
				},
			},
			"recommendations": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
			},
		},
	}
}
