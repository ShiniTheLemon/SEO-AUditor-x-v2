# AI SEO Auditor

An AI-powered SEO audit tool. Users submit a website URL, the backend generates an audit report with OpenAI, stores it in Supabase (PostgreSQL), and the frontend displays the latest reports.

## Structure

- `frontend/`: React + TypeScript SPA built with Vite. URL input UI and report dashboard.
- `backend/`: Go HTTP API (net/http). Config, PostgreSQL/Supabase storage, OpenAI report generation, and device-fingerprint report scoping.

## API

- `GET /health` — health check.
- `POST /seo-reports` — create an audit report for a URL.
- `GET /seo-reports` — list latest reports (scoped by device fingerprint).

## Target Flow

1. A user enters a website URL in the frontend.
2. The frontend sends the URL to the Go backend.
3. The Go backend fetches or analyzes the website content.
4. The Go backend calls the OpenAI API, formats the audit report, and applies report-generation logic.
5. The Go backend saves the report into the Supabase `seo_reports` table.
6. The frontend fetches and displays the latest reports.

Supabase is used only as the database layer. It should not own the audit-generation, formatting, or report-composition logic.

## Development

Backend (requires `.env` with `DATABASE_URL`):

```sh
cd backend
go run ./cmd/api
```

Frontend:

```sh
cd frontend
npm install
npm run dev
```

Run backend tests with `go test ./...` in `backend/`.
