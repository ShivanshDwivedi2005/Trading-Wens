import { describe, expect, it } from "vitest";
import { parseNewsFeed } from "./news-api";

describe("news feed contract", () => {
  it("accepts a versioned GDELT response", () => {
    const result = parseNewsFeed({
      data: [
        {
          id: "article-1",
          title: "NVIDIA expands its data center platform",
          url: "https://example.com/article",
          domain: "example.com",
          published_at: "2026-10-06T15:59:59Z",
          language: "English",
          source_country: "United States",
          matched_symbols: ["NVDA"],
        },
      ],
      as_of: "2026-10-06T16:00:00Z",
      source: "gdelt",
      count: 1,
    });
    expect(result.data[0]?.matched_symbols).toEqual(["NVDA"]);
  });

  it("rejects malformed responses", () => {
    expect(() => parseNewsFeed({ source: "gdelt" })).toThrow();
  });
});
