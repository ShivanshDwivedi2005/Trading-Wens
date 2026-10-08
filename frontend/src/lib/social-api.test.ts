import { describe, expect, it } from "vitest";
import { parseSocialFeed } from "./social-api";

describe("social feed contract", () => {
  it("accepts a normalized X response", () => {
    const result = parseSocialFeed({
      data: [
        {
          id: "199",
          text: "Apple reports stronger demand",
          url: "https://x.com/marketdesk/status/199",
          author_name: "Market Desk",
          username: "marketdesk",
          created_at: "2026-10-08T08:30:00Z",
          language: "en",
          matched_symbol: "AAPL",
          metrics: { likes: 14, replies: 2, reposts: 5, quotes: 1 },
        },
      ],
      as_of: "2026-10-08T08:31:00Z",
      source: "x",
      symbol: "AAPL",
      count: 1,
    });
    expect(result.data[0]?.username).toBe("marketdesk");
  });

  it("rejects malformed responses", () => {
    expect(() => parseSocialFeed({ source: "x" })).toThrow();
  });
});
