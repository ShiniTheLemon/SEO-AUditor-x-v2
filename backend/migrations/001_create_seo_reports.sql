create table if not exists public.seo_reports (
  id uuid primary key default gen_random_uuid(),
  url text not null,
  title text,
  summary text,
  score integer,
  report jsonb not null,
  user_id text,
  device_details jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create index if not exists seo_reports_created_at_idx
  on public.seo_reports (created_at desc);

create index if not exists seo_reports_user_created_at_idx
  on public.seo_reports (user_id, created_at desc);
