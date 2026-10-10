import { describe, expect, it } from "vitest";
import { parseAuthSession } from "./auth-api";

describe("authentication contract", () => {
  it("accepts a backend user session", () => {
    const session = parseAuthSession({
      user: {
        id: "google-user-1",
        email: "analyst@example.com",
        display_name: "Market Analyst",
        avatar_url: "https://example.com/avatar.png",
      },
    });
    expect(session.user.display_name).toBe("Market Analyst");
  });

  it("requires an authenticated user identity", () => {
    expect(() => parseAuthSession({ user: { email: "not-an-email" } })).toThrow();
  });
});
