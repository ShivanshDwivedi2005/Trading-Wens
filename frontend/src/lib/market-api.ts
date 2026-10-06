import { z } from "zod";
import { supabase } from "@/integrations/supabase/client";

const marketSnapshotSchema = z.object({
  symbol: z.string(),
  name: z.string(),
  price: z.number(),
  change: z.number(),
  change_percent: z.number(),
  open: z.number(),
  high: z.number(),
  low: z.number(),
  previous_close: z.number(),
  volume: z.number().nonnegative(),
  timestamp: z.string(),
  available: z.boolean(),
});

const marketSnapshotResponseSchema = z.object({
  data: z.array(marketSnapshotSchema),
  as_of: z.string(),
  feed: z.string(),
  source: z.literal("alpaca"),
  count: z.number().int().nonnegative(),
});

export type MarketSnapshot = z.infer<typeof marketSnapshotSchema>;
export type MarketSnapshotResponse = z.infer<typeof marketSnapshotResponseSchema>;

export function parseMarketSnapshotResponse(value: unknown): MarketSnapshotResponse {
  return marketSnapshotResponseSchema.parse(value);
}

export async function fetchMarketSnapshots(): Promise<MarketSnapshotResponse> {
  const {
    data: { session },
  } = await supabase.auth.getSession();
  if (!session?.access_token) {
    throw new Error("Your session has expired. Sign in again to load market data.");
  }

  const configuredBaseURL = import.meta.env["VITE_API_BASE_URL"]?.trim() ?? "";
  const baseURL = configuredBaseURL.replace(/\/$/, "");
  const response = await fetch(`${baseURL}/api/v1/market/snapshots`, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${session.access_token}`,
    },
  });

  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("Your session has expired. Sign in again to load market data.");
    }
    throw new Error("Live market data is temporarily unavailable.");
  }

  return parseMarketSnapshotResponse(await response.json());
}
