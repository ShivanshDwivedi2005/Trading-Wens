import { z } from "zod";
import { authenticatedFetch } from "./auth-api";

const newsArticleSchema = z.object({
  id: z.string(),
  title: z.string(),
  url: z.string().url(),
  domain: z.string(),
  published_at: z.string(),
  language: z.string(),
  source_country: z.string(),
  image_url: z.string().url().optional(),
  matched_symbols: z.array(z.string()),
});

const newsFeedSchema = z.object({
  data: z.array(newsArticleSchema),
  as_of: z.string(),
  source: z.literal("gdelt"),
  count: z.number().int().nonnegative(),
});

export type NewsArticle = z.infer<typeof newsArticleSchema>;
export type NewsFeedResponse = z.infer<typeof newsFeedSchema>;

export function parseNewsFeed(value: unknown): NewsFeedResponse {
  return newsFeedSchema.parse(value);
}

export async function fetchMarketNews(symbol?: string): Promise<NewsFeedResponse> {
  const params = symbol ? `?${new URLSearchParams({ symbol })}` : "";
  const response = await authenticatedFetch(`/api/v1/news${params}`);
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("Your session has expired. Sign in again to load market news.");
    }
    throw new Error("Market news is temporarily unavailable.");
  }
  return parseNewsFeed(await response.json());
}
