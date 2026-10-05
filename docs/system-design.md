# System design

This document specifies the durable data model, live-state model, domain contracts, confidence policy, deduplication rules, concurrency model, and external interfaces for the target system.

## Central domain contract

`RiskSignal` is the canonical result of the intelligence pipeline. In language-neutral schema notation, the public version 1 contract is:

```graphql
type RiskSignal {
  ID: string!
  Ticker: string!
  Timestamp: int64!
  Headline: string!
  Source: string!

  Sentiment: float64!
  SentimentConfidence: float64!

  Event: string!
  EventConfidence: float64!

  Relevance: float64!
  Novelty: float64!
  SourceReliability: float64!
  MarketReaction: float64!

  Impact: float64!
  RiskLevel: string!
}
```

The service-owned representation also carries fields needed for traceability and safe downstream processing: `CompanyID`, `EventClusterID`, `ArticleID`, `NewsMomentum`, `SocialMomentum`, `Actionable`, `UncertaintyReason`, `ModelVersion`, `PolicyVersion`, and `CreatedAt`. Those fields may be added to a later public contract version without changing the meaning of the version 1 fields.

`Timestamp` is Unix milliseconds in UTC. Scores other than `Impact` are normalized to `[-1, 1]` for signed sentiment and `[0, 1]` for confidence and contextual features. `Impact` is in `[1, 10]`.

## Confidence and actionability policy

Thresholds are configuration values recorded with the policy version. The initial defaults are:

- minimum sentiment confidence: `0.70`
- minimum event confidence: `0.70`
- minimum company relevance: `0.65`
- minimum actionable impact: `6.0`

If sentiment confidence is below its threshold, the signal uses `Sentiment = 0`, records an uncertainty reason, and sets `RiskLevel = UNCERTAIN`. If event confidence is below its threshold, the signal uses `Event = UNCERTAIN` and sets `RiskLevel = UNCERTAIN`. If either required prediction is uncertain, `Actionable` is false regardless of the provisional impact.

An uncertain prediction is not discarded. It is stored for calibration, monitoring, and later review, but it is not published to the rebalance or stress streams. The frontend may display it as uncertain evidence and must not style it as an actionable risk classification.

## Event clustering and momentum

Deduplication has two stages:

1. Exact or near-exact fingerprints prevent the same provider item, URL, or normalized headline/body from being processed more than once.
2. Semantic clustering groups similar coverage for the same canonical company and event type within a configurable window, initially 6 hours.

Cluster matching considers canonical company, normalized entities, event type, publication time, and headline or embedding similarity. A cluster has one representative event and one current aggregate risk evaluation.

Additional articles do not add another full impact value. They update cluster evidence:

- `news_momentum` uses the time-decayed count of independent publishers and reports.
- `social_momentum` uses the time-decayed count of independent authors or communities after spam and repost filtering.
- repeated wire copies, syndicated text, reposts, and the same author repeating a claim have sharply reduced or zero independence weight.
- corroboration can increase confidence or momentum within configured caps, but it cannot multiply the base impact by the raw article count.

The cluster ID is the idempotency boundary for downstream portfolio actions. A cluster update may create a new signal version, but the rebalancer applies cooldowns and maximum cumulative change per cluster.

## PostgreSQL: durable system of record

PostgreSQL stores business facts and history that must survive process or Redis loss.

### Companies and aliases

- `companies`: canonical company identity, ticker, exchange, sector, status, and metadata.
- `company_aliases`: maintained names, former names, abbreviations, and ticker aliases used by entity resolution.

### News, social, and clustering

- `articles`: normalized source records, canonical URL, fingerprint, timestamps, raw-reference metadata, and processing state.
- `social_posts`: normalized social records and source metadata when retention is permitted.
- `article_companies`: article-to-company mapping with entity span and relevance.
- `event_clusters`: canonical company, event type, time window, representative headline, status, and aggregate evidence.
- `event_cluster_members`: member article or post, similarity, independence weight, and membership timestamp.

### NLP and risk

- `nlp_results`: article or cluster reference, sentiment, event, base impact, confidences, model versions, request ID, and inference time.
- `risk_signals`: immutable signal versions, contextual features, impact, risk level, actionability, uncertainty reason, and policy version.
- `source_reliability_history`: effective-dated source reliability scores and the evidence or policy behind them.

### Market history

- `market_ticks` or partitioned market bars: company, provider, event time, price, size, and sequence metadata.
- `market_features`: returns, volatility, volume change, and reaction features for a defined window and calculation version.

High-volume market tables should be time-partitioned. Batch inserts use conflict handling on provider and sequence identity so retries remain safe.

### Portfolio and stress history

- `portfolios` and `portfolio_holdings`: portfolio identity, ownership, versioned holdings, and effective time.
- `rebalance_events` and `rebalance_changes`: triggering signal and cluster, constraints, before/after weights, status, rationale, and timestamps.
- `stress_scenarios`: versioned scenario definitions and shock parameters.
- `stress_test_results` and `stress_position_results`: triggering signal, portfolio snapshot, before/after valuation, loss, and position-level effects.

### Operational audit

- `provider_observations`: durable provider availability summaries when long-term service-level analysis is required.
- `processing_failures`: redacted dead-letter metadata, stage, attempts, and resolution status.

Database constraints enforce unique provider records, unique fingerprints where appropriate, monotonic signal versions within a cluster, and valid score ranges. Foreign keys connect every derived result to its evidence.

## Redis: live and ephemeral state

Key names are namespaced and versioned where a schema change is possible. Representative keys are:

```text
price:NVDA                         latest normalized market state
risk:NVDA                          latest accepted RiskSignal snapshot
article:fingerprint:<sha256>       short-lived dedupe marker
cluster:active:NVDA:<cluster-id>    active cluster working state
provider:health:alpaca              provider health and last success
ratelimit:finnhub:<window>          provider rate-limit counter
subscriptions:<client-id>           authorized WebSocket subscriptions
idempotency:<consumer>:<event-id>   short-lived consumer guard
```

The stream names are defined in [architecture.md](architecture.md). All live keys have explicit TTL or lifecycle ownership except the latest price and risk snapshots, which are overwritten and can be rebuilt from providers or PostgreSQL. Rate-limit updates use atomic Redis operations or Lua scripts where multiple values must change together.

Redis eviction is treated as loss of cache and recoverable work state, not loss of durable history. Deployment policy should use persistence appropriate for stream recovery, but PostgreSQL remains authoritative.

## Concurrency and backpressure

The Go service uses structured concurrency: every long-lived goroutine is owned by a component, accepts a cancellation context, and participates in graceful shutdown. No request or stream entry may create an unbounded detached goroutine.

Bounded worker pools are used for:

- provider ingestion and polling
- normalization and validation
- entity resolution and clustering
- NLP request batching
- PostgreSQL batch writing
- Redis Stream consumption
- WebSocket serialization and broadcasting

Each stage has a bounded input channel, configurable worker count, processing deadline, retry budget, and saturation metric. Backpressure propagates to the previous stage or Redis Stream instead of growing memory without limit.

Market ticks follow two paths. The live-state writer updates Redis immediately, while the historical writer buffers a bounded number of ticks and flushes by maximum batch size or maximum wait time. If the database is unavailable, batches retry with backoff while the stream retains recoverable work. When capacity is exhausted, the system must apply an explicit provider-specific degradation policy and emit an alert; silent loss is not acceptable.

NLP calls use bounded batches where supported, per-request deadlines, a circuit breaker, and a concurrency limit aligned with the Python service capacity. WebSocket broadcasting uses per-client bounded queues and does not let one slow client block ingestion.

## Risk aggregation

The aggregator consumes a structured NLP result, cluster evidence, source reliability, and time-aligned market features. A versioned policy converts those features into impact and risk level. The exact coefficients may evolve through evaluation, but the following invariants hold:

- missing or stale market data reduces confidence; it is never silently interpreted as no market reaction.
- source reliability affects confidence and impact within a cap; it cannot override low model confidence.
- novelty falls as evidence repeats inside a cluster.
- momentum counts independent evidence and is capped.
- risk levels are derived from impact only after the confidence gate passes.
- the input features, policy version, and output are persisted together for reproducibility.

Initial actionable risk bands are `LOW` below `3.0`, `MODERATE` from `3.0` to below `5.0`, `HIGH` from `5.0` to below `7.0`, `SEVERE` from `7.0` to below `9.0`, and `CRITICAL` at `9.0` or above. `UNCERTAIN` is a confidence state, not another impact band.

## Versioned interfaces

The backend exposes `/api/v1` REST resources and a versioned WebSocket envelope. The interface specification should live in a checked-in OpenAPI or equivalent schema before frontend and backend integration. Breaking contract changes require a new API version.

Every event envelope includes `version`, `type`, `id`, `occurred_at`, `trace_id`, and `data`. Consumers ignore unknown additive fields and reject unsupported major versions. Domain enums and timestamps use one documented representation across Go, Python, and TypeScript.

## Security and data handling

- Provider credentials, database credentials, Redis credentials, auth secrets, and model-service credentials are supplied through deployment-managed secrets.
- Logs exclude tokens, passwords, raw private portfolio content, and unrestricted article bodies.
- REST and WebSocket subscriptions enforce user and portfolio authorization.
- External text is untrusted input and is size-limited, normalized, and never used to construct executable queries.
- Retention and permitted fields for social content follow provider terms and applicable policy.
- User profiles are created by application logic on the first authenticated session; no trigger is installed in the managed authentication schema.

## Observability

Metrics cover ingestion rate and lag, provider errors and failover, stream pending depth, worker saturation, batch size and flush latency, NLP latency and confidence distribution, uncertain-signal rate, cluster size, database failures, Redis errors, WebSocket queue pressure, rebalances, and stress-test duration.

Structured traces carry one trace ID from provider input through normalization, NLP, risk aggregation, persistence, and client delivery. Health endpoints distinguish liveness from readiness and report degraded dependencies without exposing secrets.
