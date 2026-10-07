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

const marketBarSchema = z.object({
  timestamp: z.string(),
  open: z.number(),
  high: z.number(),
  low: z.number(),
  close: z.number(),
  volume: z.number().nonnegative(),
});

const stockHistorySchema = z.object({
  symbol: z.string(),
  name: z.string(),
  data: z.array(marketBarSchema),
  as_of: z.string(),
  range: z.enum(["1D", "5D", "1M"]),
  timeframe: z.enum(["5Min", "15Min", "1Hour"]),
  feed: z.string(),
  source: z.literal("alpaca"),
  count: z.number().int().nonnegative(),
});

export type MarketSnapshot = z.infer<typeof marketSnapshotSchema>;
export type MarketSnapshotResponse = z.infer<typeof marketSnapshotResponseSchema>;
export type StockHistoryRange = "1D" | "5D" | "1M";
export type StockHistory = z.infer<typeof stockHistorySchema>;

export function parseMarketSnapshotResponse(value: unknown): MarketSnapshotResponse {
  return marketSnapshotResponseSchema.parse(value);
}

export function parseStockHistory(value: unknown): StockHistory {
  return stockHistorySchema.parse(value);
}

async function getAccessToken() {
  const {
    data: { session },
  } = await supabase.auth.getSession();
  if (!session?.access_token) {
    throw new Error("Your session has expired. Sign in again to load market data.");
  }
  return session.access_token;
}

function apiBaseURL() {
  return (import.meta.env["VITE_API_BASE_URL"]?.trim() ?? "").replace(/\/$/, "");
}

export async function fetchMarketSnapshots(): Promise<MarketSnapshotResponse> {
  const accessToken = await getAccessToken();
  const response = await fetch(`${apiBaseURL()}/api/v1/market/snapshots`, {
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${accessToken}`,
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

export async function fetchStockHistory(
  symbol: string,
  historyRange: StockHistoryRange,
): Promise<StockHistory> {
  const accessToken = await getAccessToken();
  const params = new URLSearchParams({ range: historyRange });
  const response = await fetch(
    `${apiBaseURL()}/api/v1/market/stocks/${encodeURIComponent(symbol)}/history?${params}`,
    {
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${accessToken}`,
      },
    },
  );
  if (!response.ok) {
    if (response.status === 401) {
      throw new Error("Your session has expired. Sign in again to load market data.");
    }
    if (response.status === 404) {
      throw new Error("This stock is not available in the tracked market universe.");
    }
    throw new Error("Live price history is temporarily unavailable.");
  }
  return parseStockHistory(await response.json());
}
