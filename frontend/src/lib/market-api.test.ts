import { describe, expect, it } from "vitest";
import { parseMarketSnapshotResponse, parseStockHistory } from "./market-api";

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

describe("stock history contract", () => {
  it("accepts normalized Alpaca bars", () => {
    const result = parseStockHistory({
      symbol: "AAPL",
      name: "Apple",
      data: [
        {
          timestamp: "2026-10-06T15:45:00Z",
          open: 250,
          high: 253,
          low: 249.5,
          close: 252.75,
          volume: 140000,
        },
      ],
      as_of: "2026-10-06T15:45:00Z",
      range: "1D",
      timeframe: "5Min",
      feed: "iex",
      source: "alpaca",
      count: 1,
    });

    expect(result.data[0]?.close).toBe(252.75);
  });

  it("rejects an unsupported range", () => {
    expect(() =>
      parseStockHistory({
        symbol: "AAPL",
        name: "Apple",
        data: [],
        as_of: "2026-10-06T15:45:00Z",
        range: "1Y",
        timeframe: "1Day",
        feed: "iex",
        source: "alpaca",
        count: 0,
      }),
    ).toThrow();
  });
});
