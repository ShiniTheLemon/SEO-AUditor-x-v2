package database

import "database/sql"

func EnsureSchema(db *sql.DB) error {
	statements := []string{
		`create table if not exists public.seo_reports (
  id uuid primary key default gen_random_uuid(),
  url text not null,
  title text,
  summary text,
  score integer,
  report jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
)`,
		`alter table public.seo_reports add column if not exists title text`,
		`alter table public.seo_reports add column if not exists summary text`,
		`alter table public.seo_reports add column if not exists score integer`,
		`alter table public.seo_reports add column if not exists report jsonb not null default '{}'::jsonb`,
		`alter table public.seo_reports add column if not exists created_at timestamptz not null default now()`,
		`create index if not exists seo_reports_created_at_idx on public.seo_reports (created_at desc)`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}
