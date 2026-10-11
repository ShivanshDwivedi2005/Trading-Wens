import { describe, expect, it } from "vitest";
import { portfolioRisk, positionRisk } from "./portfolio-risk";
import type { PriceMovementPrediction } from "./market-prediction";
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

  it("marks a predicted drop as adverse for a long position and an increase as aligned", () => {
    const down = prediction("down");
    const up = prediction("up");
    expect(positionRisk(position, 5000, down).alignment).toBe("adverse");
    expect(positionRisk(position, 5000, up).alignment).toBe("aligned");
    expect(positionRisk(position, 5000, down).score).toBeGreaterThan(
      positionRisk(position, 5000, up).score,
    );
  });

  it("weights every position prediction into cumulative portfolio risk", () => {
    const second = { ...position, symbol: "MSFT", market_value: 3100 };
    const result = portfolioRisk(
      [position, second],
      5000,
      new Map([
        ["AAPL", prediction("down")],
        ["MSFT", prediction("up")],
      ]),
    );
    expect(result.adverseCount).toBe(1);
    expect(result.alignedCount).toBe(1);
    expect(result.score).toBeGreaterThan(0);
  });
});

function prediction(direction: "up" | "down"): PriceMovementPrediction {
  return {
    direction,
    label: direction === "up" ? "Move up" : "Move down",
    possibility: 80,
    confidence: 85,
    score: direction === "up" ? 0.6 : -0.6,
    analyzedHeadlines: 3,
    matchedHeadlines: 3,
    reason: "test prediction",
  };
}
