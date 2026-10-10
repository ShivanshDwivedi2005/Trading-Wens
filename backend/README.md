# Backend service

The Go service owns Google OAuth, signed application sessions, server-side database persistence, provider credentials, normalized market-data delivery, and model-enriched multi-provider news. The browser never receives database, Google, Alpaca, or session-signing secrets.

## Run locally

From this directory:

```sh
go run ./cmd/api
```

The service loads local settings from `../frontend/.env` and `../.env` without overriding environment variables supplied by the process. Apply `migrations/001_trading_ledger.sql` to Neon, then configure its PostgreSQL `DATABASE_URL`, Google OAuth, the application signing key, and Alpaca credentials. Start the Python model service from `../ml` so `FINBERT_INFERENCE_URL` is reachable. Bluesky public post search is enabled by default and does not require credentials. Acquisition steps for every setting are documented in `../docs/configuration.md`.

The frontend development server proxies `/api` to `http://127.0.0.1:8080`. Deployed environments can set `VITE_API_BASE_URL` when the API uses a separate origin and must include that origin in `CORS_ALLOWED_ORIGINS`.

## Implemented endpoints

- `GET /api/v1/auth/google/start`
- `GET /auth/google/callback`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`
- `GET /api/v1/market/snapshots` (authenticated)
- `GET /api/v1/market/stocks/{symbol}/history?range=1D|5D|1M` (authenticated)
- `GET /api/v1/news?symbol={symbol}` (authenticated, Alpaca + GDELT with trained sentiment, symbol optional)
- `GET /api/v1/social?symbol={symbol}` (authenticated, Bluesky public post search)
- `GET /api/v1/trading/portfolio` (authenticated, persisted Alpaca paper account snapshot)
- `GET /api/v1/trading/assets?search={query}` (authenticated, active tradable equities)
- `GET /api/v1/trading/orders` (authenticated, persisted paper orders, fills, audit trail, and sentiment evidence)
- `POST /api/v1/trading/orders` (authenticated, confirmed paper orders only)
- `DELETE /api/v1/trading/orders/{orderID}` (authenticated, confirmed working-order cancellation)
- `GET /health`

The versioned interface is documented in `../docs/api-v1.yaml`.
