# Submission datasets

The files in this directory are compact, non-proprietary samples used to make the repository and presentation reproducible without copying authenticated provider payloads.

## Files

- `synthetic-sentiment-history.csv` is a deliberately synthetic hourly series for three stocks. Values are presentation-scale sentiment indices, not model evaluation results or live market observations.
- `synthetic-risk-events.json` mirrors the shape and range of the labelled demonstration events in the frontend. Every row carries `"synthetic": true`.
- `data-sources.json` records each runtime source, its access model, the fields used by Trading Wens, and whether payloads are stored in this submission.

## Assumptions

- Sentiment ranges from `-100` to `100` in the CSV display sample.
- Impact is an explainable severity score from `1.0` to `10.0`.
- Confidence is model or signal confidence, not the probability that the stock price will move in a stated direction.
- `Sideways` means available evidence does not clear the directional threshold. A high sentiment-class confidence can still coexist with a sideways price forecast because these are different measurements.
- Runtime provider records are requested under the user's own credentials or from public endpoints and are not redistributed here.
