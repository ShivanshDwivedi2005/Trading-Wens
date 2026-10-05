# Design documentation

The target Trading Wens design is split into four documents:

- [System architecture](architecture.md): runtime topology, component boundaries, data flow, and operational guarantees.
- [System design](system-design.md): `RiskSignal`, confidence policy, event clustering, PostgreSQL and Redis ownership, concurrency, interfaces, security, and observability.
- [Runtime workflows](workflows.md): startup, market, news, NLP, portfolio, user, recovery, and shutdown flows.
- [Target repository layout](repository-layout.md): intended mature source tree and workspace rules.

These documents describe the target state. The repository may implement it incrementally, and simulated frontend data must remain visibly distinct from live provider data throughout that transition.
