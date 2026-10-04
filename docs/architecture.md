# Architecture direction

Trading Wens is organized as a monorepo with independently deployable frontend and backend boundaries.

## Frontend

`frontend/` contains the TanStack Start application. Public routes may render the shared market simulation, while authenticated workspace routes remain under the pathless `_authenticated` gate. Supabase currently provides authentication and profile persistence.

## Backend

`backend/` reserves the package boundaries for a Go service without committing to implementation details prematurely:

- `cmd/api` will hold the executable entry point.
- `internal/config` will own validated runtime configuration.
- `internal/domain` will define core news, social post, sentiment, stock, and signal concepts.
- `internal/ingestion/news` and `internal/ingestion/social` will isolate provider adapters.
- `internal/sentiment` will coordinate text preparation, model inference, confidence, and explanation metadata.
- `internal/market` will map sentiment events to stock-level signals.
- `internal/platform/database` will contain persistence adapters.
- `internal/transport/http` will expose versioned API handlers.
- `migrations` and `tests` are reserved for service-owned database changes and higher-level tests.

The backend should expose versioned contracts to the frontend. Provider payloads must be normalized at the ingestion boundary so external schemas never leak into domain or transport layers.

## Data integrity

Every signal should preserve source, publication time, ingestion time, affected ticker, model version, confidence, and explanation metadata. Simulated fixtures must remain visibly distinguishable from provider data in both APIs and the interface.

## Security baseline

Credentials belong in local or deployment-managed environment variables. Logs must not contain provider tokens, session tokens, raw passwords, or full private user content. Administrative Supabase credentials are server-only.
