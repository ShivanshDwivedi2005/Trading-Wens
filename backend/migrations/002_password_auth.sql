begin;

create table if not exists public.user_credentials (
  user_id text primary key references public.profiles(id) on delete cascade,
  username text not null check (username ~ '^[A-Za-z0-9_.-]{3,32}$'),
  email text not null,
  password_hash text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists user_credentials_username_idx
  on public.user_credentials (lower(username));
create unique index if not exists user_credentials_email_idx
  on public.user_credentials (lower(email));

commit;
