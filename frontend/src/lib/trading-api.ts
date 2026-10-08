import { z } from "zod";
import { authenticatedFetch } from "./auth-api";

const accountSchema = z.object({
  id: z.string(),
  status: z.string(),
  currency: z.string(),
  cash: z.number(),
  buying_power: z.number(),
  portfolio_value: z.number(),
  equity: z.number(),
  last_equity: z.number(),
  long_market_value: z.number(),
  trading_blocked: z.boolean(),
  pattern_day_trader: z.boolean(),
});

const positionSchema = z.object({
  symbol: z.string(),
  asset_id: z.string(),
  side: z.string(),
  quantity: z.number(),
  average_entry_price: z.number(),
  current_price: z.number(),
  market_value: z.number(),
  cost_basis: z.number(),
  unrealized_pl: z.number(),
  unrealized_pl_percent: z.number(),
  change_today: z.number(),
});

const portfolioSchema = z.object({
  account: accountSchema,
  positions: z.array(positionSchema),
  as_of: z.string(),
  source: z.literal("alpaca"),
  mode: z.literal("paper"),
});

const assetSchema = z.object({
  id: z.string(),
  symbol: z.string(),
  name: z.string(),
  exchange: z.string(),
  asset_class: z.string(),
  status: z.string(),
  tradable: z.boolean(),
  fractionable: z.boolean(),
});

const assetsSchema = z.object({
  data: z.array(assetSchema),
  count: z.number().int().nonnegative(),
  source: z.literal("alpaca"),
  mode: z.literal("paper"),
});

const orderSchema = z.object({
  id: z.string(),
  client_order_id: z.string(),
  symbol: z.string(),
  quantity: z.number(),
  filled_quantity: z.number(),
  side: z.string(),
  type: z.string(),
  time_in_force: z.string(),
  status: z.string(),
  limit_price: z.number().optional(),
  submitted_at: z.string(),
  mode: z.literal("paper"),
});

export type TradingPosition = z.infer<typeof positionSchema>;
export type TradingPortfolio = z.infer<typeof portfolioSchema>;
export type TradingAsset = z.infer<typeof assetSchema>;
export type PaperOrder = z.infer<typeof orderSchema>;

export type PaperOrderInput = {
  symbol: string;
  quantity: number;
  side: "buy" | "sell";
  type: "market" | "limit";
  limit_price?: number;
  time_in_force: "day";
};

export function parseTradingPortfolio(value: unknown): TradingPortfolio {
  return portfolioSchema.parse(value);
}

export function parsePaperOrder(value: unknown): PaperOrder {
  return orderSchema.parse(value);
}

async function authorizedFetch(path: string, init?: RequestInit) {
  const response = await authenticatedFetch(path, init);
  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as {
      error?: { message?: string };
    } | null;
    throw new Error(payload?.error?.message ?? "Alpaca trading data is temporarily unavailable.");
  }
  return response;
}

export async function fetchTradingPortfolio(): Promise<TradingPortfolio> {
  const response = await authorizedFetch("/api/v1/trading/portfolio");
  return parseTradingPortfolio(await response.json());
}

export async function fetchTradingAssets(search = "") {
  const params = new URLSearchParams();
  if (search.trim()) params.set("search", search.trim());
  const response = await authorizedFetch(`/api/v1/trading/assets?${params}`);
  return assetsSchema.parse(await response.json());
}

export async function submitPaperOrder(input: PaperOrderInput): Promise<PaperOrder> {
  const response = await authorizedFetch("/api/v1/trading/orders", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return parsePaperOrder(await response.json());
}
