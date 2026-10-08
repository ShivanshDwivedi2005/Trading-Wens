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

Configure the Google OAuth web client, provider credentials, and trained checkpoint path in `.env`. The registered Google redirect URI must exactly match `GOOGLE_REDIRECT_URI`; locally it is `http://localhost:8080/auth/google/callback`. Start the Python model service before the Go API. Credentials remain server-only, while the browser receives only a signed HTTP-only application session cookie. See [runtime configuration](docs/configuration.md) for every variable and where to obtain it.

## Quality checks

```sh
pnpm check
```

This runs linting, TypeScript validation, unit tests, and a production build.

## Current stack

- React 19 and TanStack Start
- TypeScript and Tailwind CSS
- Google OpenID Connect authentication with signed HTTP-only sessions
- Go backend with Google OAuth, Alpaca market/trading/news APIs, GDELT news, and X recent search
- Python FinBERT inference service backed by the trained Trading Wens sentiment checkpoint

Simulation fixtures live in `frontend/src/lib/market.ts`. They must remain visibly identified as illustrative data until real provider integrations are implemented.

The authenticated workspace uses the configured Alpaca paper account for its overview, position-risk feed, portfolio-risk status, searchable asset list, live candlesticks, confirmed paper orders, and news. GDELT adds global company coverage. The backend deduplicates the combined news feed and enriches every available headline with calibrated sentiment from the trained model. When `X_ENABLED=true`, the X recent-search API is the separate social source and supplies per-stock market conversation. Simulated monitoring stays visibly separated from provider data.
