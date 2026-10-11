# Repository layout

The submission layout keeps the three runtime workspaces independent while placing implementation beneath the guideline's required `src/` boundary.

```text
Trading-Wens/
├── README.md
├── LICENSE
├── package.json
├── pnpm-lock.yaml
├── requirements.txt
├── src/
│   ├── frontend/
│   │   ├── src/components/
│   │   ├── src/lib/
│   │   ├── src/routes/
│   │   ├── public/
│   │   └── package.json
│   ├── backend/
│   │   ├── cmd/api/
│   │   ├── internal/
│   │   ├── migrations/
│   │   ├── tests/
│   │   └── go.mod
│   └── ml/
│       ├── inference/
│       └── requirements.txt
├── data/
│   ├── README.md
│   ├── data-sources.json
│   ├── synthetic-risk-events.json
│   └── synthetic-sentiment-history.csv
└── docs/
    ├── presentation.pptx
    ├── presentation.pdf
    ├── architecture.png
    ├── images/
    ├── api-v1.yaml
    ├── architecture.md
    ├── configuration.md
    ├── system-design.md
    └── workflows.md
```

## Boundary rules

- `src/frontend` owns presentation, routing, shared market fixtures, and browser API clients. Authenticated pages stay beneath the pathless `_authenticated` route gate.
- `src/backend` owns sessions, provider credentials, normalized contracts, persistence, and paper-order execution. The browser never receives server credentials.
- `src/ml` owns the local trained-model inference contract. Checkpoint weights remain deployment artifacts outside Git.
- Versioned contracts live in `docs/api-v1.yaml`. Frontend and backend types are not copied ad hoc across workspace boundaries.
- `data` contains only submission-safe synthetic examples and a source register. Authenticated provider payloads are not redistributed.
- Generated output, dependency directories, credentials, local environments, model weights, and editor state remain ignored.
