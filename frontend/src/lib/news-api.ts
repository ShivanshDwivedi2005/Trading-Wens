import { z } from "zod";
import { authenticatedFetch } from "./auth-api";

const newsArticleSchema = z.object({
  id: z.string(),
  title: z.string(),
  summary: z.string().optional(),
  url: z.string().url(),
  domain: z.string(),
  provider: z.string(),
  published_at: z.string(),
  language: z.string(),
  source_country: z.string(),
  image_url: z.string().url().optional(),
  matched_symbols: z.array(z.string()),
  sentiment: z
    .object({
      label: z.enum(["NEGATIVE", "NEUTRAL", "POSITIVE"]),
      score: z.number().min(-1).max(1),
      confidence: z.number().min(0).max(1),
      uncertainty: z.number().min(0).max(1),
      probabilities: z.object({
        NEGATIVE: z.number().min(0).max(1),
        NEUTRAL: z.number().min(0).max(1),
        POSITIVE: z.number().min(0).max(1),
      }),
      model: z.string(),
      model_version: z.string(),
    })
    .optional(),
});

const newsFeedSchema = z.object({
  data: z.array(newsArticleSchema),
  as_of: z.string(),
  source: z.literal("aggregated"),
  providers: z.array(z.string()),
  count: z.number().int().nonnegative(),
});

export type NewsArticle = z.infer<typeof newsArticleSchema>;
export type NewsSentiment = NonNullable<NewsArticle["sentiment"]>;
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
