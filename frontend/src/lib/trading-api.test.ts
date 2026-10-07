import { describe, expect, it } from "vitest";
import { parsePaperOrder, parseTradingPortfolio } from "./trading-api";

describe("trading API contracts", () => {
  it("accepts normalized Alpaca paper portfolio data", () => {
    const result = parseTradingPortfolio({
      account: {
        id: "account-1",
        status: "ACTIVE",
        currency: "USD",
        cash: 10000,
        buying_power: 20000,
        portfolio_value: 25000,
        equity: 25000,
        last_equity: 24500,
        long_market_value: 15000,
        trading_blocked: false,
        pattern_day_trader: false,
      },
      positions: [],
      as_of: "2026-10-08T10:00:00Z",
      source: "alpaca",
      mode: "paper",
    });
    expect(result.account.portfolio_value).toBe(25000);
  });

  it("accepts paper order responses and rejects live mode", () => {
    const order = {
      id: "order-1",
      client_order_id: "client-1",
      symbol: "AAPL",
      quantity: 1,
      filled_quantity: 0,
      side: "buy",
      type: "market",
      time_in_force: "day",
      status: "accepted",
      submitted_at: "2026-10-08T10:00:00Z",
      mode: "paper",
    };
    expect(parsePaperOrder(order).mode).toBe("paper");
    expect(() => parsePaperOrder({ ...order, mode: "live" })).toThrow();
  });
});
