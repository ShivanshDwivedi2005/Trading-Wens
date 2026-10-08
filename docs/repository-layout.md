# Target repository layout

The following layout describes the intended mature project. Existing paths may be introduced incrementally. Directory names are architectural boundaries, not a requirement to create empty placeholders.

```text
financial-risk-intelligence/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go
│   │   ├── domain/
│   │   │   ├── market.go
│   │   │   ├── article.go
│   │   │   ├── company.go
│   │   │   ├── risk.go
│   │   │   └── portfolio.go
│   │   ├── providers/
│   │   │   ├── market/
│   │   │   │   ├── provider.go
│   │   │   │   ├── manager.go
│   │   │   │   ├── alpaca/client.go
│   │   │   │   ├── finnhub/client.go
│   │   │   │   └── twelvedata/client.go
│   │   │   └── news/
│   │   │       ├── marketaux/
│   │   │       ├── finnhub/
│   │   │       ├── alphavantage/
│   │   │       ├── gdelt/
│   │   │       ├── sec/
│   │   │       └── x/
│   │   ├── ingestion/
│   │   │   ├── market.go
│   │   │   ├── news.go
│   │   │   └── social.go
│   │   ├── normalize/
│   │   │   ├── market.go
│   │   │   └── news.go
│   │   ├── entity/
│   │   │   ├── resolver.go
│   │   │   ├── ticker_map.go
│   │   │   └── relevance.go
│   │   ├── dedupe/
│   │   │   ├── fingerprint.go
│   │   │   ├── similarity.go
│   │   │   └── cluster.go
│   │   ├── market/
│   │   │   ├── return.go
│   │   │   ├── volatility.go
│   │   │   ├── volume.go
│   │   │   └── reaction.go
│   │   ├── nlp/
│   │   │   ├── client.go
│   │   │   ├── request.go
│   │   │   └── response.go
│   │   ├── risk/
│   │   │   ├── aggregator.go
│   │   │   ├── credibility.go
│   │   │   ├── novelty.go
│   │   │   └── level.go
│   │   ├── rebalance/
│   │   │   ├── engine.go
│   │   │   ├── constraints.go
│   │   │   └── normalize.go
│   │   ├── stress/
│   │   │   ├── engine.go
│   │   │   ├── scenario.go
│   │   │   └── valuation.go
│   │   ├── queue/
│   │   │   └── redis_streams.go
│   │   ├── cache/
│   │   │   └── redis.go
│   │   ├── database/
│   │   │   ├── postgres.go
│   │   │   ├── company_repository.go
│   │   │   ├── market_repository.go
│   │   │   ├── news_repository.go
│   │   │   ├── risk_repository.go
│   │   │   └── portfolio_repository.go
│   │   ├── websocket/
│   │   │   ├── hub.go
│   │   │   ├── client.go
│   │   │   └── subscriptions.go
│   │   └── api/
│   │       ├── router.go
│   │       ├── stocks.go
│   │       ├── risk.go
│   │       ├── portfolio.go
│   │       └── stress.go
│   ├── migrations/
│   │   ├── 001_companies.sql
│   │   ├── 002_market.sql
│   │   ├── 003_news.sql
│   │   ├── 004_risk.sql
│   │   └── 005_portfolio.sql
│   └── go.mod
├── ml/
│   ├── notebooks/
│   │   ├── sentiment_training.ipynb
│   │   ├── event_training.ipynb
│   │   └── impact_training.ipynb
│   ├── datasets/
│   ├── models/
│   │   ├── sentiment_finbert/
│   │   ├── event_finbert/
│   │   └── impact_finbert/
│   ├── inference/
│   │   ├── main.py
│   │   ├── models.py
│   │   ├── sentiment.py
│   │   ├── event.py
│   │   ├── impact.py
│   │   └── schemas.py
│   ├── evaluation/
│   │   ├── sentiment_eval.py
│   │   ├── event_eval.py
│   │   └── impact_eval.py
│   └── requirements.txt
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Stock.tsx
│   │   │   ├── Index.tsx
│   │   │   └── StressTest.tsx
│   │   ├── components/
│   │   │   ├── RiskHeatmap.tsx
│   │   │   ├── MarketChart.tsx
│   │   │   ├── EventFeed.tsx
│   │   │   ├── RiskCard.tsx
│   │   │   ├── WeightChart.tsx
│   │   │   └── ProviderHealth.tsx
│   │   ├── api/
│   │   └── websocket/
│   └── package.json
├── infrastructure/
│   ├── nginx/nginx.conf
│   ├── postgres/
│   └── redis/
├── monitoring/
│   ├── prometheus.yml
│   └── grafana/
├── docs/
│   ├── architecture.md
│   ├── system-design.md
│   ├── workflows.md
│   └── repository-layout.md
├── docker-compose.yml
├── .env.example
├── .gitignore
└── README.md
```

## Boundary rules

- `backend/internal/domain` contains provider-independent business concepts.
- `backend/internal/providers` contains external schemas and adapter behavior.
- `backend/internal/database`, `cache`, and `queue` implement persistence ports; domain rules do not depend directly on vendor clients.
- `ml/inference` is the runtime Python service. Notebooks, datasets, evaluation code, and training artifacts are offline concerns.
- `frontend` consumes versioned backend contracts and does not import backend implementation types.
- Generated output, dependency directories, credentials, local environment files, downloaded datasets, and large model weights remain outside Git unless an explicit artifact policy says otherwise.
- The current TanStack route structure may differ from the illustrative `pages` tree. Authenticated workspace routes must remain beneath the pathless authentication gate.
