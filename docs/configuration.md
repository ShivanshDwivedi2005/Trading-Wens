# Runtime configuration

Copy `.env.example` to `.env` and fill only the values for services you run. Keep `.env` local; it is ignored by Git.

## Required application credentials

### Database persistence

- `DATABASE_URL`: the HTTPS project URL from Supabase **Project Settings → API**. This is the project API URL (for example `https://project-ref.supabase.co`), not a browser key and not a PostgreSQL connection string.
- `DATABASE_SECRET_KEY`: create or copy a server-side secret key from **Settings → API Keys**. A legacy `service_role` key is also accepted through the compatibility aliases, but new Supabase projects should use a secret key. Never put this value in `frontend/.env`, a `VITE_*` variable, browser code, screenshots, or Git.
- `DATABASE_REQUEST_TIMEOUT_SECONDS`: maximum duration of profile and trading-ledger writes. The default is 15 seconds.

Before starting the API, run `backend/migrations/001_trading_ledger.sql` once in the Supabase SQL Editor. The backend performs a startup check and refuses to serve authenticated trading data when the schema or database credentials are unavailable.

The migration creates Google-authenticated application profiles without modifying Supabase's managed `auth` schema. All order, fill, audit, account, and position rows are scoped by the Google user ID and provider account ID. A provider account can be owned by only one application user, preventing the globally configured paper account from being exposed through another login. Current positions are upserted, position/account history is sampled once per minute, fills are idempotent, and audit events are append-only.

The current Alpaca Trading API integration connects one paper account through server environment variables. Supporting independent Alpaca accounts for multiple application users requires a provider account-connection flow (Alpaca OAuth or Broker API); do not copy one account's API keys into multiple user records.

### Google OAuth

- `GOOGLE_OAUTH_CLIENT_ID` and `GOOGLE_OAUTH_CLIENT_SECRET`: create a Web application credential in [Google Cloud Console](https://console.cloud.google.com/apis/credentials). Add the exact URI from `GOOGLE_OAUTH_REDIRECT_URI` as an authorized redirect URI.
- `GOOGLE_OAUTH_REDIRECT_URI`: use `http://localhost:8080/auth/google/callback` locally and the HTTPS backend callback in production.
- `APP_SESSION_SIGNING_KEY`: generate at least 32 random characters. PowerShell can generate a suitable value with `[Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(48))`.

### Alpaca market, trading, and news data

- `ALPACA_API_KEY` and `ALPACA_SECRET_KEY`: create or sign in to an [Alpaca account](https://app.alpaca.markets/), select the paper-trading environment, and generate API keys. The same credentials authenticate market snapshots, bars, paper trading, asset search, and Alpaca News.
- `ALPACA_DATA_FEED`: `iex` works with the free US equities entitlement. Use another supported feed only when the Alpaca account is entitled to it.
- `ALPACA_DATA_REST_URL` and `ALPACA_TRADING_REST_URL`: keep the defaults unless an approved Alpaca-compatible deployment requires a different data host. Trading is intentionally restricted to Alpaca's paper endpoint.

### Trained sentiment model

- `NLP_MODEL_DIR`: absolute path to the trained checkpoint directory containing `model.safetensors`, tokenizer files, `labels.json`, and `calibration.json`. For the supplied training project, this is `Trading-Wens-NLP-Training/trading_wens_ml/models/sentiment/best`.
- `NLP_MODEL_VERSION`: stable deployment identifier such as the training date or model release tag. It is returned with every prediction for traceability.
- `NLP_MODEL_NAME`: human-readable model family name returned by the inference API.
- `NLP_LISTEN_ADDRESS`: host and port for the Python inference process. Keep it on loopback for local development.
- `FINBERT_INFERENCE_URL`: URL the Go API uses to reach the Python inference process.
- `NLP_MAX_BATCH_SIZE` and `NLP_REQUEST_TIMEOUT_SECONDS`: bound inference memory use and request latency. The defaults are suitable for the current news feed limits.

Install the pinned Python packages and start the model before the Go API:

```powershell
python -m venv .venv
.\.venv\Scripts\Activate.ps1
python -m pip install -r ml\requirements.txt
$env:NLP_MODEL_DIR = 'C:\path\to\Trading-Wens-NLP-Training\trading_wens_ml\models\sentiment\best'
$env:NLP_MODEL_VERSION = 'sentiment-2026-10-08'
python ml\inference\server.py
```

The model weights are not committed because they are generated deployment artifacts and exceed normal Git hosting limits.

## Keyless news source

- `GDELT_API_URL`: GDELT DOC 2.0 is public and does not require registration or an API key. The default endpoint is ready to use.

The `/api/v1/news` endpoint requests GDELT and Alpaca News concurrently, normalizes their records, removes duplicate URLs, and runs one sentiment batch across the combined feed. If one news provider fails, the healthy provider still supplies the response. If the model service fails, articles remain available without a fabricated sentiment value.

## Optional X social data

- `X_API_ENABLED`: leave `false` when X data is not configured.
- `X_API_BEARER_TOKEN`: create a developer project and app in the [X Developer Portal](https://developer.x.com/en/portal/dashboard), enable a plan with recent-search access, and copy the app bearer token.
- `X_API_BASE_URL`: keep the default unless using an approved compatible gateway.

X posts remain a separate social feed. They are not included in news sentiment until the trained model contract is explicitly extended to social content.

## Network and session settings

- `FRONTEND_URL`: public frontend origin used after OAuth.
- `BACKEND_ADDRESS`: Go server listen address.
- `CORS_ALLOWED_ORIGINS`: comma-separated frontend origins allowed to send authenticated browser requests.
- `AUTH_SESSION_TTL_SECONDS`: signed application session lifetime from 300 seconds to seven days.
- `VITE_API_BASE_URL`: optional browser-visible backend origin when frontend and backend are deployed separately.

The backend temporarily accepts the previous Google, Alpaca, NLP, X, and Supabase variable names as aliases so existing local environments keep working. New deployments should use the canonical names in `.env.example`.
