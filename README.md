# AI SEO Auditor

This repo contains planning and implementation notes for an AI SEO Auditor.

The project is split into two markdown files:

- [FRONTEND.md](FRONTEND.md): Next.js static export, React, Tailwind CSS, Supabase client, URL input UI, and dashboard display.
- [BACKEND.md](BACKEND.md): Go / Gin / GORM API, OpenAI API call, report formatting, Supabase PostgreSQL storage.

## Target Flow

1. A user enters a website URL in the frontend.
2. The frontend sends the URL to the Go backend.
3. The Go backend fetches or analyzes the website content.
4. The Go backend calls the OpenAI API, formats the audit report, and applies report-generation logic.
5. The Go backend saves the report into the Supabase `seo_reports` table.
6. The frontend fetches and displays the latest reports.

Supabase is used only as the database layer. It should not own the audit-generation, formatting, or report-composition logic.
