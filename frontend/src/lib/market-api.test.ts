import { describe, expect, it } from "vitest";
import { parseMarketSnapshotResponse } from "./market-api";

describe("market snapshot contract", () => {
  it("accepts the versioned API response", () => {
    const response = parseMarketSnapshotResponse({
      data: [
        {
          symbol: "AAPL",
          name: "Apple",
          price: 251.5,
          change: 1.5,
          change_percent: 0.6,
          open: 248,
          high: 253,
          low: 247.5,
          previous_close: 250,
          volume: 42000000,
          timestamp: "2026-10-06T15:59:59Z",
          available: true,
        },
      ],
      as_of: "2026-10-06T15:59:59Z",
      feed: "iex",
      source: "alpaca",
      count: 1,
    });

    expect(response.data[0]?.symbol).toBe("AAPL");
  });

  it("rejects incomplete provider data", () => {
    expect(() => parseMarketSnapshotResponse({ data: [] })).toThrow();
  });
});
