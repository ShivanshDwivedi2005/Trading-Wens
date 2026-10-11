import { describe, expect, it } from "vitest";
import { priceMovementPrediction } from "./market-prediction";
import type { MarketSnapshot } from "./market-api";
import type { NewsArticle } from "./news-api";

const snapshot: MarketSnapshot = {
  symbol: "AAPL",
  name: "Apple",
  price: 210,
  change: 5,
  change_percent: 2.5,
  open: 205,
  high: 212,
  low: 204,
  previous_close: 205,
  volume: 1_000_000,
  timestamp: "2026-10-11T10:00:00Z",
  available: true,
};

const positiveArticle: NewsArticle = {
  id: "news-1",
  title: "Apple growth beats expectations",
  url: "https://example.com/apple",
  domain: "example.com",
  provider: "alpaca",
  published_at: "2026-10-11T09:00:00Z",
  language: "English",
  source_country: "United States",
  matched_symbols: ["AAPL"],
  sentiment: {
    label: "POSITIVE",
    score: 0.8,
    confidence: 0.9,
    uncertainty: 0.1,
    probabilities: { NEGATIVE: 0.02, NEUTRAL: 0.08, POSITIVE: 0.9 },
    model: "trading-wens-sentiment",
    model_version: "test",
  },
};

describe("price movement prediction", () => {
  it("combines live movement and trained news sentiment", () => {
    const prediction = priceMovementPrediction(snapshot, [positiveArticle]);
    expect(prediction.direction).toBe("up");
    expect(prediction.possibility).toBeGreaterThan(50);
    expect(prediction.analyzedHeadlines).toBe(1);
  });

  it("reports matched headlines that still await model analysis", () => {
    const prediction = priceMovementPrediction(snapshot, [
      { ...positiveArticle, sentiment: undefined },
    ]);
    expect(prediction.matchedHeadlines).toBe(1);
    expect(prediction.analyzedHeadlines).toBe(0);
    expect(prediction.reason).toContain("awaiting trained-model analysis");
  });
});
