# Trading Wens

Trading Wens is a market-intelligence platform for turning financial news and social-market conversations into explainable stock sentiment signals. The current interface uses clearly labelled simulation data while the ingestion and prediction services are being designed.

## Repository layout

```text
.
├── backend/             # Go service boundary (structure only for now)
├── docs/                # Architecture notes and delivery roadmap
├── frontend/            # TanStack Start web application
├── .cursor/rules/       # Repository-specific engineering guidance
└── package.json         # Workspace commands
```

The intended backend responsibilities are separated by domain: source ingestion, normalization, sentiment analysis, market-signal generation, persistence, and HTTP delivery. See [docs/architecture.md](docs/architecture.md) for the planned boundaries.

## Local development

Requirements: Node.js 20 or newer, pnpm 11 or newer, and Go 1.23 or newer.

```sh
pnpm install
copy frontend\.env.example frontend\.env
copy .env.example .env
cd backend && go run ./cmd/api
cd ..
pnpm dev
```

Fill in the public Supabase values in `frontend/.env` and the server-side provider values in `.env`. Both files are ignored by Git. During local development the backend reads the public auth project from `frontend/.env` so it can validate the same sessions the browser creates. Alpaca and Supabase secret keys remain server-only.

## Quality checks

```sh
pnpm check
```

This runs linting, TypeScript validation, unit tests, and a production build.

## Current stack

- React 19 and TanStack Start
- TypeScript and Tailwind CSS
- Supabase authentication and PostgreSQL
- Go backend with Supabase session validation and normalized Alpaca snapshots

Simulation fixtures live in `frontend/src/lib/market.ts`. They must remain visibly identified as illustrative data until real provider integrations are implemented.

The authenticated market overview uses Alpaca data, and the live risk feed uses current GDELT coverage. Both are labelled separately from the simulated risk and sentiment features.
