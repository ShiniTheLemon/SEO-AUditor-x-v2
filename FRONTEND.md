# AI SEO Auditor Frontend

This file describes the frontend implementation for the AI SEO Auditor.

The frontend uses:

- Next.js with Static Export
- React
- Tailwind CSS
- Supabase client SDK

The frontend lets users submit a URL, calls the Go backend to generate an audit, fetches recent reports from Supabase, and displays them on a dashboard.

## Frontend Flow

1. User enters a website URL.
2. Next.js sends the URL to the Go backend API.
3. The Go backend creates, formats, and stores the SEO report.
4. The frontend fetches the latest records from the `seo_reports` table.
5. The dashboard displays recent SEO reports.

## Frontend Structure

```txt
apps/
  web/
    app/
      page.tsx
      dashboard/
        page.tsx
    components/
      UrlAuditForm.tsx
      ReportsDashboard.tsx
    lib/
      supabaseClient.ts
    next.config.js
    package.json
    tailwind.config.ts
    postcss.config.js
```

## Install Commands

```bash
cd apps/web
npm install next react react-dom @supabase/supabase-js
npm install -D typescript tailwindcss postcss autoprefixer eslint
```

## Next.js Static Export Configuration

File: `apps/web/next.config.js`

```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "export",
  images: {
    unoptimized: true,
  },
};

module.exports = nextConfig;
```

## Frontend Environment Variables

File: `apps/web/.env.local`

```env
NEXT_PUBLIC_SUPABASE_URL=your_supabase_project_url
NEXT_PUBLIC_SUPABASE_ANON_KEY=your_supabase_anon_key
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

Do not expose the OpenAI API key or Supabase service role key in frontend environment variables.

## Supabase Client

File: `apps/web/lib/supabaseClient.ts`

```ts
import { createClient } from "@supabase/supabase-js";

export const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
);
```

## URL Audit Form

File: `apps/web/components/UrlAuditForm.tsx`

```tsx
"use client";

import { useState } from "react";

type UrlAuditFormProps = {
  onReportCreated?: () => void;
};

export function UrlAuditForm({ onReportCreated }: UrlAuditFormProps) {
  const [url, setUrl] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/seo-reports`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ url }),
        },
      );

      const payload = await response.json();

      if (!response.ok) {
        throw new Error(payload.error || "Failed to generate SEO report.");
      }

      setUrl("");
      onReportCreated?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong.");
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4 rounded-lg border p-4">
      <div>
        <label htmlFor="url" className="block text-sm font-medium">
          Website URL
        </label>
        <input
          id="url"
          type="url"
          value={url}
          onChange={(event) => setUrl(event.target.value)}
          placeholder="https://example.com"
          required
          className="mt-2 w-full rounded-md border px-3 py-2"
        />
      </div>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      <button
        type="submit"
        disabled={isLoading}
        className="rounded-md bg-black px-4 py-2 text-white disabled:opacity-50"
      >
        {isLoading ? "Auditing..." : "Generate Audit"}
      </button>
    </form>
  );
}
```

## Reports Dashboard

File: `apps/web/components/ReportsDashboard.tsx`

```tsx
"use client";

import { useCallback, useEffect, useState } from "react";
import { supabase } from "../lib/supabaseClient";

type SeoReport = {
  id: string;
  url: string;
  title: string | null;
  summary: string | null;
  score: number | null;
  report: {
    checks?: Array<{
      name: string;
      status: string;
      details: string;
    }>;
    recommendations?: string[];
  };
  created_at: string;
};

export function ReportsDashboard() {
  const [reports, setReports] = useState<SeoReport[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  const fetchReports = useCallback(async () => {
    setIsLoading(true);

    const { data, error } = await supabase
      .from("seo_reports")
      .select("*")
      .order("created_at", { ascending: false })
      .limit(10);

    if (!error && data) {
      setReports(data as SeoReport[]);
    }

    setIsLoading(false);
  }, []);

  useEffect(() => {
    fetchReports();
  }, [fetchReports]);

  if (isLoading) {
    return <p className="text-sm text-gray-600">Loading reports...</p>;
  }

  return (
    <div className="space-y-4">
      {reports.map((report) => (
        <article key={report.id} className="rounded-lg border p-4">
          <div className="flex items-start justify-between gap-4">
            <div>
              <h2 className="font-semibold">{report.title || report.url}</h2>
              <p className="text-sm text-gray-600">{report.url}</p>
            </div>

            <div className="rounded-full bg-gray-100 px-3 py-1 text-sm">
              Score: {report.score ?? "N/A"}
            </div>
          </div>

          {report.summary ? (
            <p className="mt-3 text-sm">{report.summary}</p>
          ) : null}

          {report.report?.checks?.length ? (
            <div className="mt-4">
              <h3 className="text-sm font-semibold">Checks</h3>
              <ul className="mt-2 space-y-2">
                {report.report.checks.map((check) => (
                  <li key={check.name} className="text-sm">
                    <span className="font-medium">{check.name}</span>:{" "}
                    <span>{check.status}</span>
                    <p className="text-gray-600">{check.details}</p>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}
        </article>
      ))}
    </div>
  );
}
```

## Home Page

File: `apps/web/app/page.tsx`

```tsx
import { UrlAuditForm } from "../components/UrlAuditForm";
import { ReportsDashboard } from "../components/ReportsDashboard";

export default function HomePage() {
  return (
    <main className="mx-auto max-w-4xl space-y-8 p-6">
      <section>
        <h1 className="text-3xl font-bold">AI SEO Auditor</h1>
        <p className="mt-2 text-gray-600">
          Enter a URL to generate an AI-powered SEO audit report.
        </p>
      </section>

      <UrlAuditForm />

      <section className="space-y-4">
        <h2 className="text-xl font-semibold">Latest Reports</h2>
        <ReportsDashboard />
      </section>
    </main>
  );
}
```

## Frontend Notes

- Use the Supabase anon key only for browser-safe read operations.
- Keep OpenAI calls on the backend.
- Keep report creation, formatting, and audit orchestration in the Go backend.
- Use Supabase only as the database layer.
- Add client-side URL validation, but keep backend validation as the source of truth.
- For a static export, avoid server-only Next.js features unless they are replaced with client-side calls.
