# Trading Wens

Trading Wens is a market-intelligence platform for exploring live large-cap market data and current company coverage alongside clearly labelled simulated risk tooling.

## Repository layout

```text
.
├── backend/             # Go API and provider integrations
├── docs/                # Architecture notes and delivery roadmap
├── frontend/            # TanStack Start web application
├── ml/                  # Python sentiment inference service (weights stay external)
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

Apply the SQL files in `backend/migrations` in numeric order, then configure the database, application signing key, provider credentials, and trained checkpoint path in `.env`. Email/username authentication works independently of Google. To enable Google, set `GOOGLE_OAUTH_ENABLED=true`; the registered redirect URI must exactly match `GOOGLE_OAUTH_REDIRECT_URI`, locally `http://localhost:8080/auth/google/callback`. Credentials remain server-only, while the browser receives only a signed HTTP-only session cookie. See [runtime configuration](docs/configuration.md) for every variable and where to obtain it.

## Quality checks

```sh
pnpm check
```

This runs linting, TypeScript validation, unit tests, and a production build.

## Current stack

- React 19 and TanStack Start
- TypeScript and Tailwind CSS
- Email/username authentication and optional Google OpenID Connect with signed HTTP-only sessions
- Go backend with Google OAuth, Neon PostgreSQL persistence, Alpaca market/trading/news APIs, GDELT news, and Bluesky public post search
- Python FinBERT inference service backed by the trained Trading Wens sentiment checkpoint

Simulation fixtures live in `frontend/src/lib/market.ts`. They must remain visibly identified as illustrative data until real provider integrations are implemented.

The authenticated workspace uses the configured Alpaca paper account for its overview, position-risk feed, portfolio-risk status, searchable asset list, live candlesticks, confirmed paper orders, and news. Account, position, order, fill, and audit activity is persisted per signed-in user and provider account. GDELT adds global company coverage. The backend deduplicates the combined news feed and enriches every available headline with calibrated sentiment from the trained model. Bluesky's public post search supplies a separate, keyless per-stock market-conversation feed. Simulated monitoring stays visibly separated from provider data.
