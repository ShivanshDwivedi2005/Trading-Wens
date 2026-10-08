import { describe, expect, it } from "vitest";
import { portfolioRisk, positionRisk } from "./portfolio-risk";
import type { TradingPosition } from "./trading-api";

const position: TradingPosition = {
  symbol: "AAPL",
  asset_id: "asset-1",
  side: "long",
  quantity: 10,
  average_entry_price: 200,
  current_price: 190,
  market_value: 1900,
  cost_basis: 2000,
  unrealized_pl: -100,
  unrealized_pl_percent: -5,
  change_today: -3.5,
};

describe("portfolio risk heuristics", () => {
  it("uses live position inputs to produce a bounded score", () => {
    const result = positionRisk(position, 5000);
    expect(result.weight).toBeCloseTo(38);
    expect(result.score).toBeGreaterThan(1);
    expect(result.score).toBeLessThanOrEqual(10);
    expect(result.action).toBe("Review concentration");
  });

  it("summarizes concentration across positions", () => {
    const result = portfolioRisk([position], 5000);
    expect(result.concentration).toBeCloseTo(38);
    expect(result.level).not.toBe("");
  });
});
