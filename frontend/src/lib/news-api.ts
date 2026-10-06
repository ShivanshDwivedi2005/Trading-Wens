import { z } from "zod";
import { supabase } from "@/integrations/supabase/client";

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

export async function fetchMarketNews(): Promise<NewsFeedResponse> {
  const {
    data: { session },
  } = await supabase.auth.getSession();
  if (!session?.access_token) {
    throw new Error("Your session has expired. Sign in again to load market news.");
  }

  const configuredBaseURL = import.meta.env["VITE_API_BASE_URL"]?.trim() ?? "";
  const response = await fetch(`${configuredBaseURL.replace(/\/$/, "")}/api/v1/news`, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${session.access_token}`,
    },
  });
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("Your session has expired. Sign in again to load market news.");
    }
    throw new Error("Market news is temporarily unavailable.");
  }
  return parseNewsFeed(await response.json());
}
