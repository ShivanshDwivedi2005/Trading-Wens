import { z } from "zod";
import { authenticatedFetch } from "./auth-api";

const socialPostSchema = z.object({
  id: z.string(),
  text: z.string(),
  url: z.string().url(),
  author_name: z.string(),
  username: z.string(),
  avatar_url: z.string().url().optional(),
  created_at: z.string(),
  language: z.string(),
  matched_symbol: z.string(),
  metrics: z.object({
    likes: z.number().int().nonnegative(),
    replies: z.number().int().nonnegative(),
    reposts: z.number().int().nonnegative(),
    quotes: z.number().int().nonnegative(),
  }),
});

const socialFeedSchema = z.object({
  data: z.array(socialPostSchema),
  as_of: z.string(),
  source: z.literal("bluesky"),
  symbol: z.string(),
  count: z.number().int().nonnegative(),
});

export type SocialPost = z.infer<typeof socialPostSchema>;
export type SocialFeedResponse = z.infer<typeof socialFeedSchema>;

export function parseSocialFeed(value: unknown): SocialFeedResponse {
  return socialFeedSchema.parse(value);
}

export async function fetchSocialPosts(symbol: string): Promise<SocialFeedResponse> {
  const params = new URLSearchParams({ symbol });
  const response = await authenticatedFetch(`/api/v1/social?${params}`);
  if (!response.ok) {
    if (response.status === 503) {
      throw new Error("Bluesky market conversation is disabled by the server.");
    }
    throw new Error("Bluesky market conversation is temporarily unavailable.");
  }
  return parseSocialFeed(await response.json());
}
