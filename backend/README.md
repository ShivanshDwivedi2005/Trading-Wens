# Backend service

The Go service owns provider credentials, Supabase-backed session validation, and normalized market-data delivery. The browser never receives the Alpaca key or secret.

## Run locally

From this directory:

```sh
go run ./cmd/api
```

The service loads local settings from `../frontend/.env` and `../.env` without overriding environment variables supplied by the process. This keeps browser authentication and backend session validation on the same Supabase project while provider secrets remain in the repository-level file. In deployment, set `AUTH_SUPABASE_URL` and `AUTH_SUPABASE_PUBLISHABLE_KEY` explicitly. Required values are documented in the example files.

The frontend development server proxies `/api` to `http://127.0.0.1:8080`. Deployed environments can set `VITE_API_BASE_URL` when the API uses a separate origin and must include that origin in `CORS_ALLOWED_ORIGINS`.

## Implemented endpoints

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/signup`
- `GET /api/v1/market/snapshots` (authenticated)
- `GET /api/v1/market/stocks/{symbol}/history?range=1D|5D|1M` (authenticated)
- `GET /api/v1/news?symbol={symbol}` (authenticated, symbol optional)
- `GET /api/v1/trading/portfolio` (authenticated, Alpaca paper account)
- `GET /api/v1/trading/assets?search={query}` (authenticated, active tradable equities)
- `POST /api/v1/trading/orders` (authenticated, confirmed paper orders only)
- `GET /health`

The versioned interface is documented in `../docs/api-v1.yaml`.
