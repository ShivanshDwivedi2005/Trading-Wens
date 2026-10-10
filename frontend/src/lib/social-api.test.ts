import { describe, expect, it } from "vitest";
import { parseSocialFeed } from "./social-api";

describe("social feed contract", () => {
  it("accepts a normalized Bluesky response", () => {
    const result = parseSocialFeed({
      data: [
        {
          id: "at://did:plc:market/app.bsky.feed.post/3abc",
          text: "Apple reports stronger demand",
          url: "https://bsky.app/profile/marketdesk.bsky.social/post/3abc",
          author_name: "Market Desk",
          username: "marketdesk.bsky.social",
          created_at: "2026-10-08T08:30:00Z",
          language: "en",
          matched_symbol: "AAPL",
          metrics: { likes: 14, replies: 2, reposts: 5, quotes: 1 },
        },
      ],
      as_of: "2026-10-08T08:31:00Z",
      source: "bluesky",
      symbol: "AAPL",
      count: 1,
    });
    expect(result.data[0]?.username).toBe("marketdesk.bsky.social");
  });

  it("rejects malformed responses", () => {
    expect(() => parseSocialFeed({ source: "bluesky" })).toThrow();
  });
});
