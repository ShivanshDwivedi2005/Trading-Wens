import { useDeferredValue, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { formatDistanceToNowStrict } from "date-fns";
import { CheckCircle2, ExternalLink, Search, ShieldCheck } from "lucide-react";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { fetchStockHistory } from "@/lib/market-api";
import { fetchMarketNews } from "@/lib/news-api";
import { fetchSocialPosts } from "@/lib/social-api";
import {
  fetchTradingAssets,
  submitPaperOrder,
  type PaperOrderInput,
  type TradingAsset,
} from "@/lib/trading-api";

const money = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 2,
});

export function LiveTrading() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const [selected, setSelected] = useState<TradingAsset>({
    id: "",
    symbol: "AAPL",
    name: "Apple Inc.",
    exchange: "NASDAQ",
    asset_class: "us_equity",
    status: "active",
    tradable: true,
    fractionable: true,
  });
  const [side, setSide] = useState<"buy" | "sell">("buy");
  const [orderType, setOrderType] = useState<"market" | "limit">("market");
  const [quantity, setQuantity] = useState("1");
  const [limitPrice, setLimitPrice] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);

  const assets = useQuery({
    queryKey: ["trading", "assets", deferredSearch],
    queryFn: () => fetchTradingAssets(deferredSearch),
    staleTime: 60_000,
    retry: 1,
  });
  const history = useQuery({
    queryKey: ["market", "history", selected.symbol, "1D"],
    queryFn: () => fetchStockHistory(selected.symbol, "1D"),
    staleTime: 30_000,
    refetchInterval: 60_000,
    retry: 1,
  });
  const news = useQuery({
    queryKey: ["news", "stock", selected.symbol],
    queryFn: () => fetchMarketNews(selected.symbol),
    staleTime: 120_000,
    refetchInterval: 120_000,
    retry: 1,
  });
  const social = useQuery({
    queryKey: ["social", "stock", selected.symbol],
    queryFn: () => fetchSocialPosts(selected.symbol),
    staleTime: 120_000,
    refetchInterval: 120_000,
    retry: false,
  });
  const order = useMutation({
    mutationFn: submitPaperOrder,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["trading", "portfolio"] });
    },
  });

  const lastBar = history.data?.data.at(-1);
  const currentPrice = lastBar?.close ?? 0;
  const quantityNumber = Number(quantity);
  const priceForEstimate = orderType === "limit" ? Number(limitPrice) : currentPrice;
  const estimate = Number.isFinite(quantityNumber * priceForEstimate)
    ? quantityNumber * priceForEstimate
    : 0;
  const sentiment = useMemo(() => {
    const articles = news.data?.data ?? [];
    const analyzed = articles.flatMap((article) => (article.sentiment ? [article.sentiment] : []));
    const score = analyzed.length
      ? analyzed.reduce((sum, result) => sum + result.score, 0) / analyzed.length
      : 0;
    return {
      score,
      confidence: analyzed.length
        ? Math.round(
            (analyzed.reduce((sum, result) => sum + result.confidence, 0) / analyzed.length) * 100,
          )
        : 0,
      count: analyzed.length,
      action:
        score > 0.25
          ? "Research positive catalysts"
          : score < -0.25
            ? "Review downside risks"
            : "Wait for a stronger signal",
    };
  }, [news.data]);

  const orderInput: PaperOrderInput = {
    symbol: selected.symbol,
    quantity: quantityNumber,
    side,
    type: orderType,
    time_in_force: "day",
    ...(orderType === "limit" ? { limit_price: Number(limitPrice) } : {}),
  };
  const validOrder =
    quantityNumber > 0 &&
    Number.isFinite(quantityNumber) &&
    (orderType === "market" || Number(limitPrice) > 0);

  return (
    <div className="space-y-5">
      <div className="trading-layout">
        <aside className="asset-browser panel">
          <div className="label text-primary">ALPACA ASSETS</div>
          <h2 className="mt-2 text-base font-semibold">Tradable stocks</h2>
          <div className="relative mt-4">
            <Search size={14} className="absolute left-3 top-3 text-muted-foreground" />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search symbol or company"
              aria-label="Search tradable stocks"
              className="h-10 pl-9 text-xs"
            />
          </div>
          <div className="asset-list mt-4" aria-busy={assets.isFetching}>
            {assets.data?.data.map((asset) => (
              <button
                type="button"
                key={asset.id}
                className={`asset-row ${selected.symbol === asset.symbol ? "asset-row-active" : ""}`}
                onClick={() => {
                  setSelected(asset);
                  order.reset();
                }}
              >
                <span className="min-w-0 text-left">
                  <strong>{asset.symbol}</strong>
                  <small>{asset.name}</small>
                </span>
                <span className="text-[9px] text-muted-foreground">{asset.exchange}</span>
              </button>
            ))}
            {assets.isPending && <div className="h-64 animate-pulse rounded bg-secondary" />}
            {!assets.isPending && !assets.data?.data.length && (
              <div className="py-8 text-center text-xs text-muted-foreground">
                No tradable stocks match.
              </div>
            )}
          </div>
          <p className="mt-3 text-[10px] leading-4 text-muted-foreground">
            Search runs across all active US equities. Up to 100 matches are shown.
          </p>
        </aside>

        <section className="panel min-w-0">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <div className="label text-primary">LIVE CANDLESTICKS</div>
              <h2 className="mt-2 text-xl font-semibold">
                {selected.symbol}{" "}
                <span className="text-sm font-normal text-muted-foreground">{selected.name}</span>
              </h2>
            </div>
            <div className="text-right">
              <div className="text-2xl font-semibold tabular-nums">
                {currentPrice ? money.format(currentPrice) : "—"}
              </div>
              <div className="mt-1 text-[10px] text-muted-foreground">
                5-minute IEX bars · refreshes every minute
              </div>
            </div>
          </div>
          <div className="mt-5">
            {history.isPending ? (
              <div className="h-96 animate-pulse rounded bg-secondary" />
            ) : history.error ? (
              <div className="flex h-96 items-center justify-center text-center text-sm text-destructive">
                {history.error.message}
              </div>
            ) : (
              <CandlestickChart symbol={selected.symbol} data={history.data?.data ?? []} />
            )}
          </div>
          <div className="mt-3 flex flex-wrap justify-between gap-2 text-[10px] text-muted-foreground">
            <span>{history.data?.count ?? 0} provider bars</span>
            <span>
              {history.data
                ? `Last bar ${new Date(history.data.as_of).toLocaleString()}`
                : "Connecting to Alpaca"}
            </span>
          </div>
        </section>

        <aside className="order-panel panel">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="label text-primary">ORDER TICKET</div>
              <h2 className="mt-2 text-base font-semibold">Place paper order</h2>
            </div>
            <ShieldCheck size={20} className="text-primary" />
          </div>
          <div className="paper-notice mt-4">PAPER ACCOUNT · NO REAL MONEY</div>
          <div className="mt-5 grid grid-cols-2 gap-2">
            {(["buy", "sell"] as const).map((value) => (
              <Button
                key={value}
                type="button"
                variant={side === value ? "default" : "outline"}
                className="min-h-11 capitalize"
                onClick={() => setSide(value)}
              >
                {value}
              </Button>
            ))}
          </div>
          <label className="order-label mt-5" htmlFor="order-type">
            Order type
          </label>
          <select
            id="order-type"
            value={orderType}
            onChange={(event) => setOrderType(event.target.value as "market" | "limit")}
            className="order-select"
          >
            <option value="market">Market</option>
            <option value="limit">Limit</option>
          </select>
          <label className="order-label mt-4" htmlFor="order-quantity">
            Quantity
          </label>
          <Input
            id="order-quantity"
            type="number"
            min="0.0001"
            step={selected.fractionable ? "0.0001" : "1"}
            value={quantity}
            onChange={(event) => setQuantity(event.target.value)}
            className="h-11"
          />
          {orderType === "limit" && (
            <>
              <label className="order-label mt-4" htmlFor="limit-price">
                Limit price
              </label>
              <Input
                id="limit-price"
                type="number"
                min="0.01"
                step="0.01"
                value={limitPrice}
                onChange={(event) => setLimitPrice(event.target.value)}
                className="h-11"
              />
            </>
          )}
          <div className="mt-5 border-y border-border py-4 text-xs">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Estimated value</span>
              <strong className="tabular-nums">
                {estimate > 0 ? money.format(estimate) : "—"}
              </strong>
            </div>
            <div className="mt-2 flex justify-between">
              <span className="text-muted-foreground">Time in force</span>
              <span>Day</span>
            </div>
          </div>
          <Button
            className="mt-5 min-h-11 w-full"
            disabled={!validOrder || order.isPending}
            onClick={() => setConfirmOpen(true)}
          >
            {order.isPending ? "Submitting…" : `Review ${side} order`}
          </Button>
          {order.error && (
            <p role="alert" className="mt-3 text-xs leading-5 text-destructive">
              {order.error.message}
            </p>
          )}
          {order.data && (
            <div className="mt-4 rounded border border-primary/30 bg-primary/5 p-3 text-xs">
              <div className="flex items-center gap-2 text-primary">
                <CheckCircle2 size={14} /> Order {order.data.status}
              </div>
              <div className="mt-2 text-muted-foreground">
                Paper order {order.data.id.slice(0, 8)}…
              </div>
            </div>
          )}
        </aside>
      </div>

      <section className="panel">
        <div className="trading-insight-grid">
          <div>
            <div className="label text-primary">TRAINED FINBERT MODEL</div>
            <h2 className="mt-2 text-lg font-semibold">Sentiment and action context</h2>
            <div className="mt-5 flex flex-wrap gap-8">
              <Insight
                label="Sentiment score"
                value={`${sentiment.score >= 0 ? "+" : ""}${sentiment.score.toFixed(2)}`}
              />
              <Insight label="Model confidence" value={`${sentiment.confidence}%`} />
              <Insight label="Best action" value={sentiment.action} />
            </div>
            <p className="mt-4 text-[10px] leading-4 text-muted-foreground">
              Aggregated from {sentiment.count} model-scored news headlines. Bluesky posts remain
              separate provider context and are not included in this score. This is not investment
              advice.
            </p>
          </div>
          <div>
            <div className="label text-muted-foreground">LATEST NEWS</div>
            <div className="mt-3 divide-y divide-border">
              {news.data?.data.slice(0, 5).map((article) => (
                <a
                  key={article.id}
                  href={article.url}
                  target="_blank"
                  rel="noreferrer noopener"
                  className="flex min-h-14 items-center gap-3 py-2 text-xs leading-4 hover:text-primary"
                >
                  <span className="min-w-0 flex-1 line-clamp-2">{article.title}</span>
                  <ExternalLink size={12} className="shrink-0" />
                </a>
              ))}
              {!news.isPending && !news.data?.data.length && (
                <div className="py-5 text-xs text-muted-foreground">No recent headlines found.</div>
              )}
            </div>
          </div>
          <div>
            <div className="label text-muted-foreground">LATEST BLUESKY POSTS</div>
            <div className="mt-3 divide-y divide-border" aria-busy={social.isPending}>
              {social.isPending &&
                Array.from({ length: 3 }, (_, index) => (
                  <div className="py-3" key={`bluesky-loading-${index}`}>
                    <div className="h-3 animate-pulse rounded bg-secondary" />
                    <div className="mt-2 h-3 w-2/3 animate-pulse rounded bg-secondary" />
                  </div>
                ))}
              {social.data?.data.slice(0, 5).map((post) => (
                <a
                  key={post.id}
                  href={post.url}
                  target="_blank"
                  rel="noreferrer noopener"
                  className="block min-h-16 py-3 text-xs leading-4 hover:text-primary"
                >
                  <span className="line-clamp-2">{post.text}</span>
                  <span className="mt-1.5 flex items-center justify-between gap-2 text-[10px] text-muted-foreground">
                    <span>
                      {post.username ? `@${post.username}` : post.author_name || "Bluesky user"}
                    </span>
                    <time dateTime={post.created_at}>
                      {formatDistanceToNowStrict(new Date(post.created_at), { addSuffix: true })}
                    </time>
                  </span>
                </a>
              ))}
              {social.error && (
                <p role="alert" className="py-5 text-xs leading-5 text-muted-foreground">
                  {social.error.message}
                </p>
              )}
              {!social.isPending && !social.error && !social.data?.data.length && (
                <div className="py-5 text-xs text-muted-foreground">
                  No recent Bluesky posts found for {selected.symbol}.
                </div>
              )}
            </div>
          </div>
        </div>
      </section>

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Confirm paper order</AlertDialogTitle>
            <AlertDialogDescription>
              This submits a {side} {orderType} order for {quantity || "0"} shares of{" "}
              {selected.symbol} to the configured Alpaca paper account. No real money is used.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="rounded border border-border bg-secondary/40 p-4 text-sm">
            <div className="flex justify-between">
              <span>Estimated value</span>
              <strong>{estimate > 0 ? money.format(estimate) : "Market price"}</strong>
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={() => order.mutate(orderInput)}>
              Submit paper order
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function CandlestickChart({
  symbol,
  data,
}: {
  symbol: string;
  data: Array<{ timestamp: string; open: number; high: number; low: number; close: number }>;
}) {
  const candles = data.slice(-78);
  if (!candles.length)
    return (
      <div className="flex h-96 items-center justify-center text-sm text-muted-foreground">
        No bars returned for this session.
      </div>
    );
  const width = 900,
    height = 400,
    left = 58,
    right = 18,
    top = 18,
    bottom = 38;
  const plotWidth = width - left - right,
    plotHeight = height - top - bottom;
  const low = Math.min(...candles.map((item) => item.low));
  const high = Math.max(...candles.map((item) => item.high));
  const padding = Math.max((high - low) * 0.08, high * 0.001);
  const min = low - padding,
    max = high + padding;
  const y = (value: number) => top + ((max - value) / (max - min || 1)) * plotHeight;
  const step = plotWidth / candles.length;
  const bodyWidth = Math.max(2, Math.min(8, step * 0.58));
  return (
    <div className="candlestick-wrap">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        role="img"
        aria-label={`${symbol} intraday candlestick chart`}
        className="h-auto w-full"
      >
        <title>
          {symbol} candlestick chart with {candles.length} five-minute bars
        </title>
        {Array.from({ length: 5 }, (_, index) => {
          const value = min + ((max - min) * index) / 4;
          const lineY = y(value);
          return (
            <g key={value}>
              <line
                x1={left}
                x2={width - right}
                y1={lineY}
                y2={lineY}
                stroke="var(--chart-grid)"
                strokeDasharray="3 5"
              />
              <text
                x={left - 8}
                y={lineY + 4}
                textAnchor="end"
                fill="var(--muted-foreground)"
                fontSize="10"
              >
                ${value.toFixed(0)}
              </text>
            </g>
          );
        })}
        {candles.map((candle, index) => {
          const x = left + step * index + step / 2;
          const rising = candle.close >= candle.open;
          const bodyTop = y(Math.max(candle.open, candle.close));
          const bodyBottom = y(Math.min(candle.open, candle.close));
          return (
            <g key={candle.timestamp} tabIndex={0}>
              <title>
                {new Date(candle.timestamp).toLocaleTimeString()}: open {candle.open.toFixed(2)},
                high {candle.high.toFixed(2)}, low {candle.low.toFixed(2)}, close{" "}
                {candle.close.toFixed(2)}
              </title>
              <line
                x1={x}
                x2={x}
                y1={y(candle.high)}
                y2={y(candle.low)}
                stroke={rising ? "var(--chart-rise)" : "var(--chart-fall)"}
                strokeWidth="1.3"
              />
              <rect
                x={x - bodyWidth / 2}
                y={bodyTop}
                width={bodyWidth}
                height={Math.max(1.5, bodyBottom - bodyTop)}
                fill={rising ? "var(--background)" : "var(--chart-fall)"}
                stroke={rising ? "var(--chart-rise)" : "var(--chart-fall)"}
                strokeWidth="1.2"
              />
            </g>
          );
        })}
        <text x={left} y={height - 10} fill="var(--muted-foreground)" fontSize="10">
          {new Date(candles[0]!.timestamp).toLocaleTimeString([], {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </text>
        <text
          x={width - right}
          y={height - 10}
          textAnchor="end"
          fill="var(--muted-foreground)"
          fontSize="10"
        >
          {new Date(candles.at(-1)!.timestamp).toLocaleTimeString([], {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </text>
      </svg>
    </div>
  );
}

function Insight({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-semibold">{value}</div>
    </div>
  );
}
