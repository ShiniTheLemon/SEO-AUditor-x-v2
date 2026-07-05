# AI SEO Auditor Backend

This file describes the backend implementation for the AI SEO Auditor.

The backend uses:

- Go
- Gin
- GORM
- OpenAI API
- Supabase PostgreSQL

Supabase acts only as the database. The Go backend owns the complicated work: URL validation, website fetching or analysis, OpenAI calls, audit-report formatting, report composition, and inserts into Supabase PostgreSQL.

## Backend Flow

1. The frontend sends a URL to the Go backend.
2. The Go backend validates the URL.
3. The Go backend fetches or simulates the website HTML.
4. The Go backend calls the OpenAI API.
5. The Go backend normalizes, formats, and scores the SEO audit report.
6. The Go backend saves the formatted report into the Supabase `seo_reports` table.
7. The frontend reads recent reports from Supabase or from a Go read endpoint.

## Backend Structure

```txt
supabase/
  migrations/
    create_seo_reports.sql
services/
  api/
    main.go
    go.mod
    internal/
      database/
        database.go
      models/
        seo_report.go
      handlers/
        seo_reports.go
      reports/
        generator.go
```

## Supabase SQL

Create the `seo_reports` table in Supabase.

```sql
create table if not exists public.seo_reports (
  id uuid primary key default gen_random_uuid(),
  url text not null,
  title text,
  summary text,
  score integer,
  report jsonb not null,
  created_at timestamptz not null default now()
);

create index if not exists seo_reports_created_at_idx
  on public.seo_reports (created_at desc);
```

Optional Row Level Security policy for public reads.

```sql
alter table public.seo_reports enable row level security;

create policy "Allow public read access to SEO reports"
on public.seo_reports
for select
using (true);
```

For production, restrict writes so only the Go backend can insert or update reports using a server-side database connection string. Do not expose the Supabase service role key in the browser.

## Go Install Commands

```bash
cd services/api
go mod init ai-seo-auditor
go get github.com/gin-gonic/gin
go get gorm.io/gorm
go get gorm.io/driver/postgres
go get gorm.io/datatypes
go get github.com/google/uuid
```

## Go Environment Variables

```env
DATABASE_URL=postgresql://postgres:password@localhost:5432/postgres
OPENAI_API_KEY=your_openai_api_key
```

For Supabase, use the project connection string from the Supabase dashboard.

## Go Model

File: `services/api/internal/models/seo_report.go`

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type SeoReport struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	URL       string         `gorm:"not null" json:"url"`
	Title     string         `json:"title"`
	Summary   string         `json:"summary"`
	Score     int            `json:"score"`
	Report    datatypes.JSON `gorm:"type:jsonb;not null" json:"report"`
	CreatedAt time.Time      `json:"created_at"`
}

func (SeoReport) TableName() string {
	return "seo_reports"
}
```

## Go Database Connection

File: `services/api/internal/database/database.go`

```go
package database

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect() (*gorm.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
```

## Go Report Generator

File: `services/api/internal/reports/generator.go`

```go
package reports

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type AuditReport struct {
	Title           string        `json:"title"`
	Summary         string        `json:"summary"`
	Score           int           `json:"score"`
	Checks          []ReportCheck `json:"checks"`
	Recommendations []string      `json:"recommendations"`
}

type ReportCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

func Generate(urlValue string) (*AuditReport, []byte, error) {
	if !isValidURL(urlValue) {
		return nil, nil, errors.New("a valid URL is required")
	}

	html := simulateHTMLFetch(urlValue)
	report := formatReport(urlValue, html)
	raw, err := json.Marshal(report)
	if err != nil {
		return nil, nil, err
	}

	return report, raw, nil
}

func isValidURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func simulateHTMLFetch(urlValue string) string {
	return fmt.Sprintf(`
<html>
  <head>
    <title>Sample Website</title>
    <meta name="description" content="A sample website for SEO audit testing." />
  </head>
  <body>
    <h1>Sample Website</h1>
    <p>This simulated HTML belongs to %s</p>
  </body>
</html>
`, urlValue)
}

func formatReport(urlValue string, html string) AuditReport {
	return AuditReport{
		Title:   "SEO audit for " + urlValue,
		Summary: "Initial generated audit based on fetched page content.",
		Score:   78,
		Checks: []ReportCheck{
			{
				Name:    "Title tag",
				Status:  "pass",
				Details: "The page includes a title tag.",
			},
			{
				Name:    "Meta description",
				Status:  "pass",
				Details: "The page includes a meta description.",
			},
		},
		Recommendations: []string{
			"Replace the simulated HTML fetch with real page fetching.",
			"Add structured checks for headings, metadata, canonical tags, and indexability.",
		},
	}
}
```

This generator is the place to add the complex report logic: OpenAI prompting, structured parsing, scoring, remediation formatting, section ordering, and output normalization.

## Go Report Handler

File: `services/api/internal/handlers/seo_reports.go`

```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"ai-seo-auditor/internal/models"
	"ai-seo-auditor/internal/reports"
)

type CreateSeoReportRequest struct {
	URL string `json:"url"`
}

func CreateSeoReport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request CreateSeoReportRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "A valid JSON body is required."})
			return
		}

		report, rawReport, err := reports.Generate(request.URL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		record := models.SeoReport{
			URL:     request.URL,
			Title:   report.Title,
			Summary: report.Summary,
			Score:   report.Score,
			Report:  datatypes.JSON(rawReport),
		}

		if err := db.Create(&record).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, record)
	}
}

func ListSeoReports(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var reports []models.SeoReport

		if err := db.Order("created_at desc").Limit(10).Find(&reports).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, reports)
	}
}
```

## Go Server

File: `services/api/main.go`

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"ai-seo-auditor/internal/database"
	"ai-seo-auditor/internal/handlers"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.POST("/seo-reports", handlers.CreateSeoReport(db))
	router.GET("/seo-reports", handlers.ListSeoReports(db))

	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
```

## Backend Notes

- Supabase is the database only.
- The Go backend owns report creation, formatting, scoring, and audit orchestration.
- Do not expose backend secrets to the browser.
- Keep URL validation in the Go backend.
- Replace the simulated HTML fetch with a real fetch once timeout, redirect, robots, and blocked-site behavior are defined.
- Consider storing raw HTML only if there is a clear product requirement. It can be large and may contain sensitive content.
- Add authentication and user ownership before storing private reports.
