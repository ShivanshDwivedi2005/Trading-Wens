import { describe, expect, it } from "vitest";
import { parseNewsFeed } from "./news-api";

describe("news feed contract", () => {
  it("accepts aggregated provider news with model sentiment", () => {
    const result = parseNewsFeed({
      data: [
        {
          id: "article-1",
          title: "NVIDIA expands its data center platform",
          url: "https://example.com/article",
          domain: "example.com",
          provider: "alpaca",
          published_at: "2026-10-06T15:59:59Z",
          language: "English",
          source_country: "United States",
          matched_symbols: ["NVDA"],
          sentiment: {
            label: "POSITIVE",
            score: 0.72,
            confidence: 0.88,
            uncertainty: 0.31,
            probabilities: { NEGATIVE: 0.05, NEUTRAL: 0.07, POSITIVE: 0.88 },
            model: "trading-wens-sentiment",
            model_version: "2026-10-08",
          },
        },
      ],
      as_of: "2026-10-06T16:00:00Z",
      source: "aggregated",
      providers: ["alpaca", "gdelt"],
      count: 1,
    });
    expect(result.data[0]?.matched_symbols).toEqual(["NVDA"]);
  });

  it("rejects malformed responses", () => {
    expect(() => parseNewsFeed({ source: "aggregated" })).toThrow();
  });
});
