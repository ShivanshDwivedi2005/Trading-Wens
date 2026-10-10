import { useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { formatDistanceToNowStrict } from "date-fns";
import {
  CheckCircle2,
  Crosshair,
  ExternalLink,
  RotateCcw,
  Search,
  ShieldCheck,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
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
import { fetchMarketSnapshots, fetchStockHistory, type StockHistoryRange } from "@/lib/market-api";
import { fetchMarketNews } from "@/lib/news-api";
import { fetchSocialPosts } from "@/lib/social-api";
import { fetchOrderMonitor, submitPaperOrder, type PaperOrderInput } from "@/lib/trading-api";

const money = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 2,
});

export function LiveTrading() {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [selectedSymbol, setSelectedSymbol] = useState("AAPL");
  const [historyRange, setHistoryRange] = useState<StockHistoryRange>("1D");
  const [side, setSide] = useState<"buy" | "sell">("buy");
  const [orderType, setOrderType] = useState<"market" | "limit">("market");
  const [quantity, setQuantity] = useState("1");
  const [limitPrice, setLimitPrice] = useState("");
  const [confirmOpen, setConfirmOpen] = useState(false);

  const snapshots = useQuery({
    queryKey: ["market", "sp500-top-30"],
    queryFn: fetchMarketSnapshots,
    staleTime: 10_000,
    refetchInterval: 15_000,
    retry: 1,
  });
  const trackedStocks = useMemo(() => {
    const query = search.trim().toLowerCase();
    const stocks = snapshots.data?.data ?? [];
    if (!query) return stocks;
    return stocks.filter((stock) => `${stock.symbol} ${stock.name}`.toLowerCase().includes(query));
  }, [search, snapshots.data]);
  const selected = snapshots.data?.data.find((stock) => stock.symbol === selectedSymbol) ?? {
    symbol: selectedSymbol,
    name: selectedSymbol === "AAPL" ? "Apple" : selectedSymbol,
    price: 0,
    change: 0,
    change_percent: 0,
    open: 0,
    high: 0,
    low: 0,
    previous_close: 0,
    volume: 0,
    timestamp: "",
    available: false,
  };
  const history = useQuery({
    queryKey: ["market", "history", selected.symbol, historyRange],
    queryFn: () => fetchStockHistory(selected.symbol, historyRange),
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
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["trading", "orders"] }),
        queryClient.invalidateQueries({ queryKey: ["trading", "portfolio"] }),
      ]);
    },
  });
  const execution = useQuery({
    queryKey: ["trading", "orders"],
    queryFn: fetchOrderMonitor,
    enabled: Boolean(order.data),
    staleTime: 1_000,
    refetchInterval: (query) => (query.state.data?.working_count ? 1_500 : false),
    retry: 1,
  });

  const lastBar = history.data?.data.at(-1);
  const currentPrice = selected.price || lastBar?.close || 0;
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
  const monitoredOrder = order.data
    ? (execution.data?.orders.find((item) => item.id === order.data.id) ?? order.data)
    : null;
  const limitPriceNumber = Number(limitPrice);
  const limitIsMarketable =
    orderType === "limit" &&
    currentPrice > 0 &&
    limitPriceNumber > 0 &&
    (side === "buy" ? currentPrice <= limitPriceNumber : currentPrice >= limitPriceNumber);

  return (
    <div className="space-y-5">
      <div className="trading-layout">
        <aside className="asset-browser panel">
          <div className="label text-primary">TOP 30 LIVE MARKET</div>
          <h2 className="mt-2 text-base font-semibold">Tracked large-cap stocks</h2>
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
          <div className="asset-list mt-4" aria-busy={snapshots.isFetching}>
            {trackedStocks.map((stock) => (
              <button
                type="button"
                key={stock.symbol}
                className={`asset-row ${selected.symbol === stock.symbol ? "asset-row-active" : ""}`}
                onClick={() => {
                  setSelectedSymbol(stock.symbol);
                  order.reset();
                }}
                aria-pressed={selected.symbol === stock.symbol}
              >
                <span className="min-w-0 text-left">
                  <strong>{stock.symbol}</strong>
                  <small>{stock.name}</small>
                </span>
                <span className="text-right text-[9px] tabular-nums">
                  <span className="block text-foreground">
                    {stock.available ? money.format(stock.price) : "Unavailable"}
                  </span>
                  <span className={stock.change_percent >= 0 ? "text-profit" : "text-loss"}>
                    {stock.available
                      ? `${stock.change_percent >= 0 ? "+" : ""}${stock.change_percent.toFixed(2)}%`
                      : "—"}
                  </span>
                </span>
              </button>
            ))}
            {snapshots.isPending && <div className="h-64 animate-pulse rounded bg-secondary" />}
            {!snapshots.isPending && trackedStocks.length === 0 && (
              <div className="py-8 text-center text-xs text-muted-foreground">
                No tracked stock matches your search.
              </div>
            )}
          </div>
          <p className="mt-3 text-[10px] leading-4 text-muted-foreground">
            All 30 maintained large-cap stocks stay available. Search by ticker or company name.
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
            <div className="text-left sm:text-right">
              <div className="text-2xl font-semibold tabular-nums">
                {currentPrice ? money.format(currentPrice) : "—"}
              </div>
              <div
                className={`mt-1 text-[10px] ${selected.change_percent >= 0 ? "text-profit" : "text-loss"}`}
              >
                {selected.available
                  ? `${selected.change_percent >= 0 ? "+" : ""}${selected.change_percent.toFixed(2)}% today`
                  : "Waiting for live quote"}
              </div>
            </div>
          </div>
          <div className="mt-4 flex flex-wrap items-center justify-between gap-3 border-y border-border py-3">
            <div className="flex rounded-md border border-border p-1" aria-label="Chart range">
              {(["1D", "5D", "1M"] as const).map((range) => (
                <Button
                  key={range}
                  type="button"
                  variant="ghost"
                  size="sm"
                  className={`min-h-9 px-3 text-xs ${historyRange === range ? "bg-secondary text-foreground" : "text-muted-foreground"}`}
                  aria-pressed={historyRange === range}
                  onClick={() => setHistoryRange(range)}
                >
                  {range}
                </Button>
              ))}
            </div>
            <span className="text-[10px] text-muted-foreground">
              Scroll or use +/− to zoom · drag to pan · click to mark a price
            </span>
          </div>
          <div className="mt-5">
            {history.isPending ? (
              <div className="h-96 animate-pulse rounded bg-secondary" />
            ) : history.error ? (
              <div className="flex h-96 items-center justify-center text-center text-sm text-destructive">
                {history.error.message}
              </div>
            ) : (
              <CandlestickChart
                key={`${selected.symbol}-${historyRange}`}
                symbol={selected.symbol}
                timeframe={history.data?.timeframe ?? "5Min"}
                data={history.data?.data ?? []}
              />
            )}
          </div>
          <div className="mt-3 flex flex-wrap justify-between gap-2 text-[10px] text-muted-foreground">
            <span>
              {history.data?.count ?? 0} provider bars · {history.data?.timeframe ?? "—"} · IEX
            </span>
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
              <h2 className="mt-2 text-base font-semibold">Place simulation order</h2>
            </div>
            <ShieldCheck size={20} className="text-primary" />
          </div>
          <div className="paper-notice mt-4">SIMULATION ACCOUNT · NO REAL MONEY</div>
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
            step="0.0001"
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
              <p
                className={`mt-2 text-[10px] leading-4 ${limitIsMarketable ? "text-profit" : "text-muted-foreground"}`}
              >
                {limitIsMarketable
                  ? "Marketable now — eligible for an immediate simulated fill at the best available quote."
                  : side === "buy"
                    ? "Fills when the best available price reaches this level or lower."
                    : "Fills when the best available price reaches this level or higher."}
              </p>
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
          {monitoredOrder && (
            <div className="mt-4 rounded border border-primary/30 bg-primary/5 p-3 text-xs">
              <div className="flex items-center gap-2 text-primary">
                <CheckCircle2 size={14} /> Simulation order {monitoredOrder.status}
              </div>
              <div className="mt-2 text-muted-foreground">
                {monitoredOrder.working
                  ? "Watching the live quote every 1.5 seconds for fill eligibility."
                  : monitoredOrder.filled_quantity > 0
                    ? `${monitoredOrder.filled_quantity} share${monitoredOrder.filled_quantity === 1 ? "" : "s"} filled${monitoredOrder.filled_average_price ? ` at ${money.format(monitoredOrder.filled_average_price)}` : ""}.`
                    : `Order ${monitoredOrder.id.slice(0, 8)}… is no longer working.`}
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
            <AlertDialogTitle>Confirm simulation order</AlertDialogTitle>
            <AlertDialogDescription>
              This submits a {side} {orderType} order for {quantity || "0"} shares of{" "}
              {selected.symbol} to the configured Alpaca simulation account. No real money is used.
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
              Submit simulation order
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function CandlestickChart({
  symbol,
  timeframe,
  data,
}: {
  symbol: string;
  timeframe: string;
  data: Array<{
    timestamp: string;
    open: number;
    high: number;
    low: number;
    close: number;
    volume: number;
  }>;
}) {
  const svgRef = useRef<SVGSVGElement>(null);
  const dragRef = useRef<{ x: number; offset: number; moved: boolean } | null>(null);
  const [visibleCount, setVisibleCount] = useState(Math.max(1, Math.min(78, data.length)));
  const [offset, setOffset] = useState(0);
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const [markedPrice, setMarkedPrice] = useState<number | null>(null);

  const width = 980,
    height = 470,
    left = 72,
    right = 76,
    top = 28,
    bottom = 52;
  const plotWidth = width - left - right,
    plotHeight = height - top - bottom;
  const minVisible = Math.min(12, Math.max(1, data.length));
  const maxVisible = Math.min(500, Math.max(1, data.length));
  const effectiveVisible = Math.max(minVisible, Math.min(visibleCount, maxVisible));
  const maxOffset = Math.max(0, data.length - effectiveVisible);
  const effectiveOffset = Math.max(0, Math.min(offset, maxOffset));
  const endIndex = data.length - effectiveOffset;
  const startIndex = Math.max(0, endIndex - effectiveVisible);
  const candles = data.slice(startIndex, endIndex);

  if (!data.length)
    return (
      <div className="flex h-96 items-center justify-center text-sm text-muted-foreground">
        No bars returned for this session.
      </div>
    );
  const low = Math.min(...candles.map((item) => item.low));
  const high = Math.max(...candles.map((item) => item.high));
  const padding = Math.max((high - low) * 0.1, high * 0.001);
  const min = low - padding,
    max = high + padding;
  const y = (value: number) => top + ((max - value) / (max - min || 1)) * plotHeight;
  const step = plotWidth / candles.length;
  const bodyWidth = Math.max(2, Math.min(12, step * 0.62));
  const hovered = hoveredIndex === null ? null : candles[hoveredIndex];

  function zoom(delta: number) {
    setVisibleCount((current) => Math.max(minVisible, Math.min(maxVisible, current + delta)));
    setHoveredIndex(null);
  }

  function chartPoint(clientX: number, clientY: number) {
    const bounds = svgRef.current?.getBoundingClientRect();
    if (!bounds) return null;
    return {
      x: ((clientX - bounds.left) / bounds.width) * width,
      y: ((clientY - bounds.top) / bounds.height) * height,
    };
  }

  return (
    <div className="candlestick-wrap">
      <div className="chart-toolbar">
        <div className="chart-readout" aria-live="polite">
          {hovered ? (
            <>
              <strong>{formatChartTime(hovered.timestamp)}</strong>
              <span>O {hovered.open.toFixed(2)}</span>
              <span>H {hovered.high.toFixed(2)}</span>
              <span>L {hovered.low.toFixed(2)}</span>
              <span>C {hovered.close.toFixed(2)}</span>
              <span>Vol {hovered.volume.toLocaleString()}</span>
            </>
          ) : (
            <span>Move across the chart for OHLC and volume</span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-9"
            aria-label="Zoom in chart"
            title="Zoom in"
            disabled={effectiveVisible <= minVisible}
            onClick={() => zoom(-Math.max(2, Math.round(effectiveVisible * 0.2)))}
          >
            <ZoomIn size={15} />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-9"
            aria-label="Zoom out chart"
            title="Zoom out"
            disabled={effectiveVisible >= maxVisible}
            onClick={() => zoom(Math.max(2, Math.round(effectiveVisible * 0.2)))}
          >
            <ZoomOut size={15} />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-9"
            aria-label="Reset chart view"
            title="Reset view"
            onClick={() => {
              setVisibleCount(Math.min(78, data.length));
              setOffset(0);
              setHoveredIndex(null);
            }}
          >
            <RotateCcw size={15} />
          </Button>
          <Button
            type="button"
            variant={markedPrice === null ? "ghost" : "outline"}
            size="sm"
            className="min-h-9 gap-1.5 px-2 text-[10px]"
            aria-label={markedPrice === null ? "Click chart to add price line" : "Clear price line"}
            onClick={() => setMarkedPrice(null)}
          >
            <Crosshair size={14} />
            {markedPrice === null ? "Price line" : `Clear ${money.format(markedPrice)}`}
          </Button>
        </div>
      </div>
      <svg
        ref={svgRef}
        viewBox={`0 0 ${width} ${height}`}
        role="img"
        tabIndex={0}
        aria-label={`${symbol} interactive candlestick chart. Use plus and minus to zoom, drag to pan, arrow keys to inspect candles, and click to mark a price.`}
        className="h-auto w-full touch-none select-none"
        onWheel={(event) => {
          event.preventDefault();
          zoom((event.deltaY > 0 ? 1 : -1) * Math.max(2, Math.round(effectiveVisible * 0.12)));
        }}
        onPointerDown={(event) => {
          dragRef.current = { x: event.clientX, offset: effectiveOffset, moved: false };
          event.currentTarget.setPointerCapture(event.pointerId);
        }}
        onPointerMove={(event) => {
          const point = chartPoint(event.clientX, event.clientY);
          if (point && point.x >= left && point.x <= width - right) {
            const index = Math.max(
              0,
              Math.min(candles.length - 1, Math.floor((point.x - left) / step)),
            );
            setHoveredIndex(index);
          }
          if (dragRef.current) {
            const distance = event.clientX - dragRef.current.x;
            if (Math.abs(distance) > 3) dragRef.current.moved = true;
            setOffset(
              Math.max(
                0,
                Math.min(
                  maxOffset,
                  dragRef.current.offset + Math.round((distance / plotWidth) * effectiveVisible),
                ),
              ),
            );
          }
        }}
        onPointerUp={(event) => {
          const drag = dragRef.current;
          dragRef.current = null;
          if (drag?.moved) return;
          const point = chartPoint(event.clientX, event.clientY);
          if (!point || point.y < top || point.y > height - bottom) return;
          const price = max - ((point.y - top) / plotHeight) * (max - min);
          setMarkedPrice(price);
        }}
        onPointerLeave={() => {
          dragRef.current = null;
          setHoveredIndex(null);
        }}
        onKeyDown={(event) => {
          if (event.key === "+" || event.key === "=") {
            event.preventDefault();
            zoom(-Math.max(2, Math.round(effectiveVisible * 0.2)));
          } else if (event.key === "-") {
            event.preventDefault();
            zoom(Math.max(2, Math.round(effectiveVisible * 0.2)));
          } else if (event.key === "ArrowLeft") {
            event.preventDefault();
            setHoveredIndex((current) => Math.max(0, (current ?? candles.length) - 1));
          } else if (event.key === "ArrowRight") {
            event.preventDefault();
            setHoveredIndex((current) => Math.min(candles.length - 1, (current ?? -1) + 1));
          } else if (event.key === "Escape") {
            setHoveredIndex(null);
            setMarkedPrice(null);
          }
        }}
      >
        <title>
          {symbol} candlestick chart with {candles.length} visible {timeframe} bars
        </title>
        {Array.from({ length: 6 }, (_, index) => {
          const value = min + ((max - min) * index) / 5;
          const lineY = y(value);
          return (
            <g key={value}>
              <line
                x1={left}
                x2={width - right + 8}
                y1={lineY}
                y2={lineY}
                stroke="var(--chart-grid)"
                strokeDasharray="3 5"
              />
              <text
                x={width - right + 14}
                y={lineY + 4}
                textAnchor="start"
                fill="var(--muted-foreground)"
                fontSize="11"
              >
                ${value.toFixed(value < 10 ? 3 : 2)}
              </text>
            </g>
          );
        })}
        {Array.from({ length: Math.min(6, candles.length) }, (_, tick) => {
          const index = Math.round(
            (tick * (candles.length - 1)) / Math.max(1, Math.min(6, candles.length) - 1),
          );
          const candle = candles[index]!;
          const x = left + step * index + step / 2;
          return (
            <g key={`time-${candle.timestamp}`}>
              <line
                x1={x}
                x2={x}
                y1={top}
                y2={height - bottom}
                stroke="var(--chart-grid)"
                opacity="0.45"
              />
              <text
                x={x}
                y={height - 18}
                textAnchor="middle"
                fill="var(--muted-foreground)"
                fontSize="10"
              >
                {formatAxisTime(candle.timestamp, timeframe)}
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
            <g key={candle.timestamp}>
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
        {hovered && hoveredIndex !== null && (
          <g pointerEvents="none">
            <line
              x1={left + step * hoveredIndex + step / 2}
              x2={left + step * hoveredIndex + step / 2}
              y1={top}
              y2={height - bottom}
              stroke="var(--foreground)"
              strokeDasharray="3 4"
              opacity="0.65"
            />
            <line
              x1={left}
              x2={width - right}
              y1={y(hovered.close)}
              y2={y(hovered.close)}
              stroke="var(--foreground)"
              strokeDasharray="3 4"
              opacity="0.65"
            />
          </g>
        )}
        {markedPrice !== null && markedPrice >= min && markedPrice <= max && (
          <g pointerEvents="none">
            <line
              x1={left}
              x2={width - right}
              y1={y(markedPrice)}
              y2={y(markedPrice)}
              stroke="var(--primary)"
              strokeWidth="1.5"
              strokeDasharray="7 4"
            />
            <rect
              x={width - right + 7}
              y={y(markedPrice) - 11}
              width="66"
              height="22"
              rx="3"
              fill="var(--primary)"
            />
            <text
              x={width - right + 40}
              y={y(markedPrice) + 4}
              textAnchor="middle"
              fill="var(--primary-foreground)"
              fontSize="10"
              fontWeight="600"
            >
              ${markedPrice.toFixed(2)}
            </text>
          </g>
        )}
      </svg>
      <div className="chart-status">
        <span>
          {candles.length} of {data.length} bars visible
        </span>
        <span>
          {effectiveOffset > 0
            ? `${effectiveOffset} newer bars to the right`
            : "Latest bar in view"}
        </span>
      </div>
      <details className="chart-data-table">
        <summary>Accessible OHLC data for visible candles</summary>
        <div className="mt-3 max-h-64 overflow-auto">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Open</th>
                <th>High</th>
                <th>Low</th>
                <th>Close</th>
                <th>Volume</th>
              </tr>
            </thead>
            <tbody>
              {candles.map((candle) => (
                <tr key={`row-${candle.timestamp}`}>
                  <td>{formatChartTime(candle.timestamp)}</td>
                  <td>{candle.open.toFixed(2)}</td>
                  <td>{candle.high.toFixed(2)}</td>
                  <td>{candle.low.toFixed(2)}</td>
                  <td>{candle.close.toFixed(2)}</td>
                  <td>{candle.volume.toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </div>
  );
}

function formatChartTime(timestamp: string) {
  return new Date(timestamp).toLocaleString([], {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatAxisTime(timestamp: string, timeframe: string) {
  const date = new Date(timestamp);
  return timeframe === "5Min"
    ? date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : date.toLocaleDateString([], { month: "short", day: "numeric" });
}

function Insight({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-semibold">{value}</div>
    </div>
  );
}
