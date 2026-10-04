# Backend service

This directory is reserved for the Trading Wens Go backend. It currently defines package boundaries only; no backend implementation has been added.

The planned service will ingest financial news and social-market conversations, normalize provider data, calculate explainable sentiment, derive stock signals, and expose versioned APIs to the frontend.

Implementation should begin with a short architecture decision covering the HTTP framework, job processing model, database ownership, provider retry policy, and model-serving approach. Package responsibilities are outlined in `../docs/architecture.md`.
