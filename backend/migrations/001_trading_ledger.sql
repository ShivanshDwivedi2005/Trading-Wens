begin;

create table if not exists public.profiles (
  id text primary key,
  email text not null,
  display_name text not null default '',
  avatar_url text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.broker_accounts (
  user_id text not null references public.profiles(id) on delete cascade,
  provider text not null,
  provider_account_id text not null,
  mode text not null,
  status text not null,
  currency text not null,
  cash numeric not null default 0,
  buying_power numeric not null default 0,
  portfolio_value numeric not null default 0,
  equity numeric not null default 0,
  last_equity numeric not null default 0,
  long_market_value numeric not null default 0,
  trading_blocked boolean not null default false,
  pattern_day_trader boolean not null default false,
  provider_as_of timestamptz not null,
  updated_at timestamptz not null default now(),
  primary key (user_id, provider, provider_account_id),
  unique (provider, provider_account_id)
);

create table if not exists public.account_snapshots (
  user_id text not null,
  provider text not null,
  provider_account_id text not null,
  captured_at timestamptz not null,
  cash numeric not null,
  buying_power numeric not null,
  portfolio_value numeric not null,
  equity numeric not null,
  long_market_value numeric not null,
  primary key (user_id, provider, provider_account_id, captured_at),
  foreign key (user_id, provider, provider_account_id)
    references public.broker_accounts(user_id, provider, provider_account_id) on delete cascade
);

create table if not exists public.positions (
  user_id text not null,
  provider text not null,
  provider_account_id text not null,
  symbol text not null,
  asset_id text not null,
  side text not null,
  quantity numeric not null,
  average_entry_price numeric not null,
  current_price numeric not null,
  market_value numeric not null,
  cost_basis numeric not null,
  unrealized_pl numeric not null,
  unrealized_pl_percent numeric not null,
  change_today numeric not null,
  provider_as_of timestamptz not null,
  updated_at timestamptz not null default now(),
  primary key (user_id, provider, provider_account_id, symbol),
  foreign key (user_id, provider, provider_account_id)
    references public.broker_accounts(user_id, provider, provider_account_id) on delete cascade
);

create table if not exists public.position_snapshots (
  user_id text not null,
  provider text not null,
  provider_account_id text not null,
  symbol text not null,
  captured_at timestamptz not null,
  quantity numeric not null,
  current_price numeric not null,
  market_value numeric not null,
  unrealized_pl numeric not null,
  unrealized_pl_percent numeric not null,
  change_today numeric not null,
  primary key (user_id, provider, provider_account_id, symbol, captured_at),
  foreign key (user_id, provider, provider_account_id)
    references public.broker_accounts(user_id, provider, provider_account_id) on delete cascade
);

create table if not exists public.orders (
  user_id text not null,
  provider text not null,
  provider_account_id text not null,
  provider_order_id text not null,
  client_order_id text not null,
  symbol text not null,
  side text not null,
  order_type text not null,
  time_in_force text not null,
  status text not null,
  quantity numeric not null,
  filled_quantity numeric not null,
  filled_average_price numeric,
  limit_price numeric,
  working boolean not null,
  submitted_at timestamptz not null,
  provider_updated_at timestamptz not null,
  filled_at timestamptz,
  canceled_at timestamptz,
  expired_at timestamptz,
  failed_at timestamptz,
  sentiment text,
  sentiment_score numeric,
  signal_confidence numeric,
  price_direction text,
  signal_possibility integer,
  analyzed_headlines integer,
  signal_reason text,
  signal_sources jsonb not null default '[]'::jsonb,
  updated_at timestamptz not null default now(),
  primary key (user_id, provider, provider_account_id, provider_order_id),
  foreign key (user_id, provider, provider_account_id)
    references public.broker_accounts(user_id, provider, provider_account_id) on delete cascade
);

create table if not exists public.fills (
  user_id text not null,
  provider text not null,
  provider_account_id text not null,
  provider_fill_id text not null,
  provider_order_id text not null,
  symbol text not null,
  side text not null,
  quantity numeric not null,
  cumulative_quantity numeric not null,
  leaves_quantity numeric not null,
  price numeric not null,
  fill_type text not null,
  transaction_time timestamptz not null,
  created_at timestamptz not null default now(),
  primary key (user_id, provider, provider_account_id, provider_fill_id),
  foreign key (user_id, provider, provider_account_id)
    references public.broker_accounts(user_id, provider, provider_account_id) on delete cascade
);

create table if not exists public.order_audit_events (
  user_id text not null,
  provider text not null,
  provider_account_id text not null,
  event_id text not null,
  provider_order_id text not null,
  symbol text not null,
  event_time timestamptz not null,
  event_type text not null,
  status text not null,
  side text not null,
  quantity numeric not null,
  filled_quantity numeric not null,
  price numeric,
  message text not null,
  event_source text not null,
  created_at timestamptz not null default now(),
  primary key (user_id, provider, provider_account_id, event_id),
  foreign key (user_id, provider, provider_account_id)
    references public.broker_accounts(user_id, provider, provider_account_id) on delete cascade
);

create index if not exists orders_account_status_idx
  on public.orders(user_id, provider_account_id, working, provider_updated_at desc);
create index if not exists fills_account_time_idx
  on public.fills(user_id, provider_account_id, transaction_time desc);
create index if not exists audit_account_time_idx
  on public.order_audit_events(user_id, provider_account_id, event_time desc);
create index if not exists position_snapshots_account_time_idx
  on public.position_snapshots(user_id, provider_account_id, captured_at desc);

alter table public.profiles enable row level security;
alter table public.broker_accounts enable row level security;
alter table public.account_snapshots enable row level security;
alter table public.positions enable row level security;
alter table public.position_snapshots enable row level security;
alter table public.orders enable row level security;
alter table public.fills enable row level security;
alter table public.order_audit_events enable row level security;

create or replace function public.sync_trading_state(
  p_user_id text,
  p_portfolio jsonb,
  p_monitor jsonb default '{}'::jsonb
) returns void
language plpgsql
security definer
set search_path = public
as $$
declare
  v_account jsonb := p_portfolio->'account';
  v_provider text := coalesce(nullif(p_portfolio->>'source', ''), 'alpaca');
  v_account_id text := v_account->>'id';
  v_mode text := coalesce(nullif(p_portfolio->>'mode', ''), 'paper');
  v_as_of timestamptz := coalesce((p_portfolio->>'as_of')::timestamptz, now());
  v_snapshot_at timestamptz := date_trunc('minute', v_as_of);
  v_item jsonb;
  v_signal jsonb;
begin
  if p_user_id is null or p_user_id = '' or v_account_id is null or v_account_id = '' then
    raise exception 'user and provider account are required';
  end if;

  insert into public.broker_accounts (
    user_id, provider, provider_account_id, mode, status, currency, cash, buying_power,
    portfolio_value, equity, last_equity, long_market_value, trading_blocked,
    pattern_day_trader, provider_as_of, updated_at
  ) values (
    p_user_id, v_provider, v_account_id, v_mode, v_account->>'status', v_account->>'currency',
    coalesce((v_account->>'cash')::numeric, 0), coalesce((v_account->>'buying_power')::numeric, 0),
    coalesce((v_account->>'portfolio_value')::numeric, 0), coalesce((v_account->>'equity')::numeric, 0),
    coalesce((v_account->>'last_equity')::numeric, 0), coalesce((v_account->>'long_market_value')::numeric, 0),
    coalesce((v_account->>'trading_blocked')::boolean, false),
    coalesce((v_account->>'pattern_day_trader')::boolean, false), v_as_of, now()
  ) on conflict (user_id, provider, provider_account_id) do update set
    mode = excluded.mode, status = excluded.status, currency = excluded.currency,
    cash = excluded.cash, buying_power = excluded.buying_power,
    portfolio_value = excluded.portfolio_value, equity = excluded.equity,
    last_equity = excluded.last_equity, long_market_value = excluded.long_market_value,
    trading_blocked = excluded.trading_blocked, pattern_day_trader = excluded.pattern_day_trader,
    provider_as_of = excluded.provider_as_of, updated_at = now();

  insert into public.account_snapshots (
    user_id, provider, provider_account_id, captured_at, cash, buying_power,
    portfolio_value, equity, long_market_value
  ) values (
    p_user_id, v_provider, v_account_id, v_snapshot_at,
    coalesce((v_account->>'cash')::numeric, 0), coalesce((v_account->>'buying_power')::numeric, 0),
    coalesce((v_account->>'portfolio_value')::numeric, 0), coalesce((v_account->>'equity')::numeric, 0),
    coalesce((v_account->>'long_market_value')::numeric, 0)
  ) on conflict do nothing;

  delete from public.positions
  where user_id = p_user_id and provider = v_provider and provider_account_id = v_account_id;

  for v_item in select value from jsonb_array_elements(coalesce(p_portfolio->'positions', '[]'::jsonb)) loop
    insert into public.positions (
      user_id, provider, provider_account_id, symbol, asset_id, side, quantity,
      average_entry_price, current_price, market_value, cost_basis, unrealized_pl,
      unrealized_pl_percent, change_today, provider_as_of, updated_at
    ) values (
      p_user_id, v_provider, v_account_id, v_item->>'symbol', v_item->>'asset_id', v_item->>'side',
      (v_item->>'quantity')::numeric, (v_item->>'average_entry_price')::numeric,
      (v_item->>'current_price')::numeric, (v_item->>'market_value')::numeric,
      (v_item->>'cost_basis')::numeric, (v_item->>'unrealized_pl')::numeric,
      (v_item->>'unrealized_pl_percent')::numeric, (v_item->>'change_today')::numeric,
      v_as_of, now()
    );
    insert into public.position_snapshots (
      user_id, provider, provider_account_id, symbol, captured_at, quantity, current_price,
      market_value, unrealized_pl, unrealized_pl_percent, change_today
    ) values (
      p_user_id, v_provider, v_account_id, v_item->>'symbol', v_snapshot_at,
      (v_item->>'quantity')::numeric, (v_item->>'current_price')::numeric,
      (v_item->>'market_value')::numeric, (v_item->>'unrealized_pl')::numeric,
      (v_item->>'unrealized_pl_percent')::numeric, (v_item->>'change_today')::numeric
    ) on conflict do nothing;
  end loop;

  for v_item in select value from jsonb_array_elements(coalesce(p_monitor->'orders', '[]'::jsonb)) loop
    v_signal := v_item->'signal';
    insert into public.orders (
      user_id, provider, provider_account_id, provider_order_id, client_order_id, symbol,
      side, order_type, time_in_force, status, quantity, filled_quantity,
      filled_average_price, limit_price, working, submitted_at, provider_updated_at,
      filled_at, canceled_at, expired_at, failed_at, sentiment, sentiment_score,
      signal_confidence, price_direction, signal_possibility, analyzed_headlines,
      signal_reason, signal_sources, updated_at
    ) values (
      p_user_id, v_provider, v_account_id, v_item->>'id', v_item->>'client_order_id', v_item->>'symbol',
      v_item->>'side', v_item->>'type', v_item->>'time_in_force', v_item->>'status',
      (v_item->>'quantity')::numeric, (v_item->>'filled_quantity')::numeric,
      nullif(v_item->>'filled_average_price', '')::numeric, nullif(v_item->>'limit_price', '')::numeric,
      (v_item->>'working')::boolean, (v_item->>'submitted_at')::timestamptz,
      (v_item->>'updated_at')::timestamptz, nullif(v_item->>'filled_at', '')::timestamptz,
      nullif(v_item->>'canceled_at', '')::timestamptz, nullif(v_item->>'expired_at', '')::timestamptz,
      nullif(v_item->>'failed_at', '')::timestamptz, v_signal->>'sentiment',
      nullif(v_signal->>'sentiment_score', '')::numeric, nullif(v_signal->>'confidence', '')::numeric,
      v_signal->>'price_direction', nullif(v_signal->>'possibility', '')::integer,
      nullif(v_signal->>'analyzed_headlines', '')::integer, v_signal->>'reason',
      coalesce(v_signal->'sources', '[]'::jsonb), now()
    ) on conflict (user_id, provider, provider_account_id, provider_order_id) do update set
      status = excluded.status, filled_quantity = excluded.filled_quantity,
      filled_average_price = excluded.filled_average_price, limit_price = excluded.limit_price,
      working = excluded.working, provider_updated_at = excluded.provider_updated_at,
      filled_at = excluded.filled_at, canceled_at = excluded.canceled_at,
      expired_at = excluded.expired_at, failed_at = excluded.failed_at,
      sentiment = excluded.sentiment, sentiment_score = excluded.sentiment_score,
      signal_confidence = excluded.signal_confidence, price_direction = excluded.price_direction,
      signal_possibility = excluded.signal_possibility, analyzed_headlines = excluded.analyzed_headlines,
      signal_reason = excluded.signal_reason, signal_sources = excluded.signal_sources, updated_at = now();
  end loop;

  for v_item in select value from jsonb_array_elements(coalesce(p_monitor->'fills', '[]'::jsonb)) loop
    insert into public.fills (
      user_id, provider, provider_account_id, provider_fill_id, provider_order_id, symbol,
      side, quantity, cumulative_quantity, leaves_quantity, price, fill_type, transaction_time
    ) values (
      p_user_id, v_provider, v_account_id, v_item->>'id', v_item->>'order_id', v_item->>'symbol',
      v_item->>'side', (v_item->>'quantity')::numeric, (v_item->>'cumulative_quantity')::numeric,
      (v_item->>'leaves_quantity')::numeric, (v_item->>'price')::numeric,
      v_item->>'type', (v_item->>'transaction_time')::timestamptz
    ) on conflict do nothing;
  end loop;

  for v_item in select value from jsonb_array_elements(coalesce(p_monitor->'audit_trail', '[]'::jsonb)) loop
    insert into public.order_audit_events (
      user_id, provider, provider_account_id, event_id, provider_order_id, symbol,
      event_time, event_type, status, side, quantity, filled_quantity, price, message, event_source
    ) values (
      p_user_id, v_provider, v_account_id, v_item->>'id', v_item->>'order_id', v_item->>'symbol',
      (v_item->>'timestamp')::timestamptz, v_item->>'event', v_item->>'status', v_item->>'side',
      (v_item->>'quantity')::numeric, (v_item->>'filled_quantity')::numeric,
      nullif(v_item->>'price', '')::numeric, v_item->>'message', v_item->>'source'
    ) on conflict do nothing;
  end loop;
end;
$$;

create or replace function public.get_trading_state(
  p_user_id text,
  p_provider text,
  p_account_id text
) returns jsonb
language plpgsql
security definer
set search_path = public
as $$
declare
  v_account public.broker_accounts%rowtype;
  v_positions jsonb;
  v_orders jsonb;
  v_fills jsonb;
  v_audit jsonb;
begin
  select * into v_account
  from public.broker_accounts
  where user_id = p_user_id and provider = p_provider and provider_account_id = p_account_id;
  if not found then
    raise exception 'trading account was not found';
  end if;

  select coalesce(jsonb_agg(jsonb_build_object(
    'symbol', symbol, 'asset_id', asset_id, 'side', side, 'quantity', quantity,
    'average_entry_price', average_entry_price, 'current_price', current_price,
    'market_value', market_value, 'cost_basis', cost_basis, 'unrealized_pl', unrealized_pl,
    'unrealized_pl_percent', unrealized_pl_percent, 'change_today', change_today
  ) order by abs(market_value) desc), '[]'::jsonb) into v_positions
  from public.positions
  where user_id = p_user_id and provider = p_provider and provider_account_id = p_account_id;

  select coalesce(jsonb_agg(jsonb_build_object(
    'id', provider_order_id, 'client_order_id', client_order_id, 'symbol', symbol,
    'quantity', quantity, 'filled_quantity', filled_quantity,
    'filled_average_price', filled_average_price, 'side', side, 'type', order_type,
    'time_in_force', time_in_force, 'status', status, 'limit_price', limit_price,
    'submitted_at', submitted_at, 'updated_at', provider_updated_at, 'filled_at', filled_at,
    'canceled_at', canceled_at, 'expired_at', expired_at, 'failed_at', failed_at,
    'working', working, 'mode', v_account.mode,
    'signal', case when sentiment is null then null else jsonb_build_object(
      'sentiment', sentiment, 'sentiment_score', sentiment_score,
      'confidence', signal_confidence, 'price_direction', price_direction,
      'possibility', signal_possibility, 'analyzed_headlines', analyzed_headlines,
      'reason', signal_reason, 'sources', signal_sources
    ) end
  ) order by provider_updated_at desc), '[]'::jsonb) into v_orders
  from public.orders
  where user_id = p_user_id and provider = p_provider and provider_account_id = p_account_id;

  select coalesce(jsonb_agg(jsonb_build_object(
    'id', provider_fill_id, 'order_id', provider_order_id, 'symbol', symbol,
    'side', side, 'quantity', quantity, 'cumulative_quantity', cumulative_quantity,
    'leaves_quantity', leaves_quantity, 'price', price, 'type', fill_type,
    'transaction_time', transaction_time, 'source', p_provider, 'mode', v_account.mode
  ) order by transaction_time desc), '[]'::jsonb) into v_fills
  from public.fills
  where user_id = p_user_id and provider = p_provider and provider_account_id = p_account_id;

  select coalesce(jsonb_agg(jsonb_build_object(
    'id', event_id, 'order_id', provider_order_id, 'symbol', symbol,
    'timestamp', event_time, 'event', event_type, 'status', status, 'side', side,
    'quantity', quantity, 'filled_quantity', filled_quantity, 'price', price,
    'message', message, 'source', event_source
  ) order by event_time desc), '[]'::jsonb) into v_audit
  from public.order_audit_events
  where user_id = p_user_id and provider = p_provider and provider_account_id = p_account_id;

  return jsonb_build_object(
    'portfolio', jsonb_build_object(
      'account', jsonb_build_object(
        'id', v_account.provider_account_id, 'status', v_account.status,
        'currency', v_account.currency, 'cash', v_account.cash,
        'buying_power', v_account.buying_power, 'portfolio_value', v_account.portfolio_value,
        'equity', v_account.equity, 'last_equity', v_account.last_equity,
        'long_market_value', v_account.long_market_value,
        'trading_blocked', v_account.trading_blocked,
        'pattern_day_trader', v_account.pattern_day_trader
      ),
      'positions', v_positions, 'as_of', v_account.provider_as_of,
      'source', v_account.provider, 'mode', v_account.mode
    ),
    'monitor', jsonb_build_object(
      'orders', v_orders, 'fills', v_fills, 'audit_trail', v_audit,
      'as_of', v_account.provider_as_of, 'source', v_account.provider, 'mode', v_account.mode,
      'order_count', jsonb_array_length(v_orders),
      'working_count', (select count(*) from public.orders where user_id = p_user_id and provider = p_provider and provider_account_id = p_account_id and working),
      'fill_count', jsonb_array_length(v_fills)
    )
  );
end;
$$;

revoke all on public.profiles from anon, authenticated;
revoke all on public.broker_accounts from anon, authenticated;
revoke all on public.account_snapshots from anon, authenticated;
revoke all on public.positions from anon, authenticated;
revoke all on public.position_snapshots from anon, authenticated;
revoke all on public.orders from anon, authenticated;
revoke all on public.fills from anon, authenticated;
revoke all on public.order_audit_events from anon, authenticated;
revoke all on function public.sync_trading_state(text, jsonb, jsonb) from public, anon, authenticated;
revoke all on function public.get_trading_state(text, text, text) from public, anon, authenticated;
grant execute on function public.sync_trading_state(text, jsonb, jsonb) to service_role;
grant execute on function public.get_trading_state(text, text, text) to service_role;

commit;
