# Backend service

The Go service owns Google OAuth, signed application sessions, provider credentials, normalized market-data delivery, and model-enriched multi-provider news. The browser never receives Google, Alpaca, X, or session-signing secrets.

## Run locally

From this directory:

```sh
go run ./cmd/api
```

The service loads local settings from `../frontend/.env` and `../.env` without overriding environment variables supplied by the process. Configure `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`, a strong `JWT_ACCESS_SECRET`, and Alpaca credentials. Start the Python model service from `../ml` so `NLP_API_URL` is reachable. To load live X posts, also set `X_ENABLED=true` and `X_BEARER_TOKEN`. Acquisition steps for every setting are documented in `../docs/configuration.md`.

The frontend development server proxies `/api` to `http://127.0.0.1:8080`. Deployed environments can set `VITE_API_BASE_URL` when the API uses a separate origin and must include that origin in `CORS_ALLOWED_ORIGINS`.

## Implemented endpoints

- `GET /api/v1/auth/google/start`
- `GET /auth/google/callback`
- `GET /api/v1/auth/session`
- `POST /api/v1/auth/logout`
- `GET /api/v1/market/snapshots` (authenticated)
- `GET /api/v1/market/stocks/{symbol}/history?range=1D|5D|1M` (authenticated)
- `GET /api/v1/news?symbol={symbol}` (authenticated, Alpaca + GDELT with trained sentiment, symbol optional)
- `GET /api/v1/social?symbol={symbol}` (authenticated, X recent search)
- `GET /api/v1/trading/portfolio` (authenticated, Alpaca paper account)
- `GET /api/v1/trading/assets?search={query}` (authenticated, active tradable equities)
- `POST /api/v1/trading/orders` (authenticated, confirmed paper orders only)
- `GET /health`

The versioned interface is documented in `../docs/api-v1.yaml`.
