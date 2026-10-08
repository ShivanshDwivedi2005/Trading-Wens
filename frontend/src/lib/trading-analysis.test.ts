import { describe, expect, it } from "vitest";
import { fillMarkDelta, positionOutlook } from "./trading-analysis";
import type { NewsArticle } from "./news-api";
import type { PaperFill, TradingPosition } from "./trading-api";

const position: TradingPosition = {
  symbol: "AAPL",
  asset_id: "asset-1",
  side: "long",
  quantity: 10,
  average_entry_price: 190,
  current_price: 205,
  market_value: 2050,
  cost_basis: 1900,
  unrealized_pl: 150,
  unrealized_pl_percent: 7.89,
  change_today: 2.1,
};

const article: NewsArticle = {
  id: "news-1",
  title: "Apple services growth beats estimates",
  url: "https://example.com/apple",
  domain: "example.com",
  provider: "alpaca",
  published_at: "2026-10-09T10:00:00Z",
  language: "English",
  source_country: "United States",
  matched_symbols: ["AAPL"],
  sentiment: {
    label: "POSITIVE",
    score: 0.75,
    confidence: 0.9,
    uncertainty: 0.2,
    probabilities: { NEGATIVE: 0.04, NEUTRAL: 0.06, POSITIVE: 0.9 },
    model: "trading-wens-sentiment",
    model_version: "test",
  },
};

it("builds a sourced profit outlook from position and news evidence", () => {
  const result = positionOutlook(position, [article]);
  expect(result.direction).toBe("profit");
  expect(result.possibility).toBeGreaterThan(50);
  expect(result.confidence).toBeGreaterThan(50);
  expect(result.sources).toHaveLength(2);
});

it("calculates a signed mark delta for buy and sell fills", () => {
  const fill = {
    id: "fill-1",
    order_id: "order-1",
    symbol: "AAPL",
    side: "buy",
    quantity: 2,
    cumulative_quantity: 2,
    leaves_quantity: 0,
    price: 200,
    type: "fill",
    transaction_time: "2026-10-09T10:00:00Z",
    source: "alpaca",
    mode: "paper",
  } satisfies PaperFill;
  expect(fillMarkDelta(fill, 205)).toBe(10);
  expect(fillMarkDelta({ ...fill, side: "sell" }, 205)).toBe(-10);
});
