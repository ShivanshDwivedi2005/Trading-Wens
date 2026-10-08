import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { formatDistanceToNowStrict } from "date-fns";
import {
  Activity,
  ArrowLeft,
  ArrowUpRight,
  ExternalLink,
  Newspaper,
  RefreshCw,
  WifiOff,
} from "lucide-react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Button } from "@/components/ui/button";
import { fetchMarketSnapshots, fetchStockHistory, type StockHistoryRange } from "@/lib/market-api";
import { fetchMarketNews } from "@/lib/news-api";
import { SentimentBadge } from "./SentimentBadge";

const ranges: StockHistoryRange[] = ["1D", "5D", "1M"];

const priceFormatter = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
});

const volumeFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

function chartLabel(timestamp: string, range: StockHistoryRange) {
  const date = new Date(timestamp);
  if (range === "1D") {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  if (range === "5D") {
    return date.toLocaleDateString([], { weekday: "short", hour: "2-digit", minute: "2-digit" });
  }
  return date.toLocaleDateString([], { month: "short", day: "numeric" });
}

export function StockDetailPage({ symbol }: { symbol: string }) {
  const [historyRange, setHistoryRange] = useState<StockHistoryRange>("1D");
  const snapshotsQuery = useQuery({
    queryKey: ["market", "sp500-top-30"],
    queryFn: fetchMarketSnapshots,
    staleTime: 10_000,
    refetchInterval: 15_000,
    retry: 1,
  });
  const historyQuery = useQuery({
    queryKey: ["market", "history", symbol, historyRange],
    queryFn: () => fetchStockHistory(symbol, historyRange),
    staleTime: 30_000,
    refetchInterval: 60_000,
    retry: 1,
  });
  const newsQuery = useQuery({
    queryKey: ["news", "stock", symbol],
    queryFn: () => fetchMarketNews(symbol),
    staleTime: 120_000,
    refetchInterval: 120_000,
    retry: 1,
  });

  const snapshot = snapshotsQuery.data?.data.find((item) => item.symbol === symbol);
  const chartData = useMemo(() => {
    const bars = [...(historyQuery.data?.data ?? [])];
    const lastBar = bars.at(-1);
    if (
      snapshot?.available &&
      (!lastBar || new Date(snapshot.timestamp).getTime() > new Date(lastBar.timestamp).getTime())
    ) {
      bars.push({
        timestamp: snapshot.timestamp,
        open: snapshot.price,
        high: snapshot.price,
        low: snapshot.price,
        close: snapshot.price,
        volume: 0,
      });
    }
    return bars.map((bar) => ({
      ...bar,
      label: chartLabel(bar.timestamp, historyRange),
    }));
  }, [historyQuery.data, historyRange, snapshot]);
  const positive = (snapshot?.change ?? 0) >= 0;
  const isRefreshing = snapshotsQuery.isFetching || historyQuery.isFetching || newsQuery.isFetching;
  const refreshAll = () => {
    void Promise.all([snapshotsQuery.refetch(), historyQuery.refetch(), newsQuery.refetch()]);
  };

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="stock-header">
        <Link to="/dashboard" className="brand flex items-center gap-2.5">
          <span className="brand-mark">
            <Activity size={17} strokeWidth={2.5} />
          </span>
          <span>
            Trading <span className="text-primary">wens</span>
          </span>
        </Link>
        <div className="flex items-center gap-2">
          <span className="hidden items-center gap-2 text-[11px] font-medium text-primary sm:flex">
            <span className="live-pulse size-1.5 rounded-full bg-primary" /> ALPACA CONNECTED
          </span>
          <Button
            variant="outline"
            size="icon"
            className="size-11"
            onClick={refreshAll}
            disabled={isRefreshing}
            aria-label="Refresh stock data and news"
          >
            <RefreshCw size={16} className={isRefreshing ? "animate-spin" : ""} />
          </Button>
        </div>
      </header>

      <main className="stock-content">
        <Link
          to="/dashboard"
          className="inline-flex min-h-11 items-center gap-2 text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft size={15} /> Back to market overview
        </Link>

        <section className="mt-5 flex flex-col justify-between gap-6 border-b border-border pb-7 md:flex-row md:items-end">
          <div>
            <div className="label flex items-center gap-2 text-primary">
              <span className="live-pulse size-1.5 rounded-full bg-primary" /> LIVE MARKET DETAIL
            </div>
            <div className="mt-3 flex flex-wrap items-baseline gap-x-4 gap-y-2">
              <h1 className="text-4xl font-semibold sm:text-5xl">{symbol}</h1>
              <span className="text-sm text-muted-foreground">
                {snapshot?.name ?? historyQuery.data?.name ?? "Tracked company"}
              </span>
            </div>
          </div>
          <div className="md:text-right">
            {snapshot?.available ? (
              <>
                <div className="text-3xl font-semibold tabular-nums sm:text-4xl">
                  {priceFormatter.format(snapshot.price)}
                </div>
                <div
                  className={`mt-2 inline-flex items-center gap-1 text-sm tabular-nums ${positive ? "text-primary" : "text-destructive"}`}
                >
                  <ArrowUpRight
                    size={15}
                    className={positive ? "" : "rotate-90"}
                    aria-hidden="true"
                  />
                  {positive ? "+" : ""}
                  {snapshot.change.toFixed(2)} ({positive ? "+" : ""}
                  {snapshot.change_percent.toFixed(2)}%) today
                </div>
              </>
            ) : snapshotsQuery.isPending ? (
              <div className="h-12 w-40 animate-pulse rounded bg-secondary" />
            ) : (
              <p className="text-sm text-muted-foreground">Current quote unavailable</p>
            )}
          </div>
        </section>

        <div className="stock-layout mt-6">
          <section className="panel min-w-0" aria-busy={historyQuery.isPending}>
            <div className="panel-heading flex-wrap gap-4">
              <div>
                <div className="label text-muted-foreground">ALPACA PRICE HISTORY</div>
                <h2 className="mt-2 text-lg font-semibold">Price movement</h2>
                <p className="mt-2 text-xs leading-5 text-muted-foreground">
                  Provider bars refresh every minute. The latest quote above refreshes every 15
                  seconds.
                </p>
              </div>
              <div className="flex rounded-md border border-border p-1" aria-label="Chart range">
                {ranges.map((range) => (
                  <Button
                    key={range}
                    variant="ghost"
                    className={`min-h-11 px-4 text-xs ${historyRange === range ? "bg-secondary text-foreground" : "text-muted-foreground"}`}
                    onClick={() => setHistoryRange(range)}
                    aria-pressed={historyRange === range}
                  >
                    {range}
                  </Button>
                ))}
              </div>
            </div>

            {historyQuery.error ? (
              <ProviderError
                message={
                  historyQuery.error instanceof Error
                    ? historyQuery.error.message
                    : "Live price history is unavailable."
                }
                onRetry={() => void historyQuery.refetch()}
              />
            ) : historyQuery.isPending ? (
              <div className="mt-7 h-80 animate-pulse rounded-md bg-secondary/70" />
            ) : chartData.length === 0 ? (
              <div className="mt-7 flex h-80 items-center justify-center rounded-md border border-border text-sm text-muted-foreground">
                Alpaca returned no bars for this range.
              </div>
            ) : (
              <div
                className="mt-7 h-80 w-full"
                role="img"
                aria-label={`${symbol} ${historyRange} closing price chart`}
              >
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart
                    data={chartData}
                    margin={{ top: 8, right: 8, left: -10, bottom: 0 }}
                    accessibilityLayer
                  >
                    <defs>
                      <linearGradient id="stockPriceFill" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="var(--primary)" stopOpacity={0.28} />
                        <stop offset="100%" stopColor="var(--primary)" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid
                      vertical={false}
                      stroke="var(--chart-grid)"
                      strokeDasharray="3 5"
                    />
                    <XAxis
                      dataKey="label"
                      tickLine={false}
                      axisLine={false}
                      minTickGap={42}
                      tick={{ fill: "var(--muted-foreground)", fontSize: 10 }}
                    />
                    <YAxis
                      domain={["auto", "auto"]}
                      tickLine={false}
                      axisLine={false}
                      width={58}
                      tickFormatter={(value: number) => `$${value.toFixed(0)}`}
                      tick={{ fill: "var(--muted-foreground)", fontSize: 10 }}
                    />
                    <Tooltip
                      cursor={{ stroke: "var(--border)" }}
                      contentStyle={{
                        background: "var(--popover)",
                        border: "1px solid var(--border)",
                        borderRadius: "var(--radius)",
                        fontSize: 12,
                      }}
                      formatter={(value) => [priceFormatter.format(Number(value)), "Close"]}
                    />
                    <Area
                      type="monotone"
                      dataKey="close"
                      stroke="var(--primary)"
                      strokeWidth={2}
                      fill="url(#stockPriceFill)"
                      isAnimationActive={false}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            )}

            <div className="mt-5 flex flex-wrap justify-between gap-2 text-[11px] text-muted-foreground">
              <span>
                {historyQuery.data
                  ? `${historyQuery.data.count} bars · ${historyQuery.data.timeframe} · ${historyQuery.data.feed.toUpperCase()} feed`
                  : "Connecting to Alpaca"}
              </span>
              <span>
                {historyQuery.data
                  ? `Last bar ${new Date(historyQuery.data.as_of).toLocaleString()}`
                  : "Waiting for price history"}
              </span>
            </div>
          </section>

          <aside className="grid content-start gap-3 sm:grid-cols-2 lg:grid-cols-1">
            <Metric
              label="OPEN"
              value={snapshot?.open ? priceFormatter.format(snapshot.open) : "—"}
            />
            <Metric
              label="DAY RANGE"
              value={
                snapshot?.low && snapshot?.high
                  ? `${priceFormatter.format(snapshot.low)} – ${priceFormatter.format(snapshot.high)}`
                  : "—"
              }
            />
            <Metric
              label="PREVIOUS CLOSE"
              value={
                snapshot?.previous_close ? priceFormatter.format(snapshot.previous_close) : "—"
              }
            />
            <Metric
              label="VOLUME"
              value={snapshot?.volume ? volumeFormatter.format(snapshot.volume) : "—"}
            />
          </aside>
        </div>

        <section className="panel mt-6" aria-busy={newsQuery.isPending}>
          <div className="panel-heading flex-wrap gap-4">
            <div>
              <div className="label flex items-center gap-2 text-primary">
                <Newspaper size={14} /> LIVE MULTI-SOURCE COVERAGE
              </div>
              <h2 className="mt-2 text-lg font-semibold">Latest {symbol} news</h2>
              <p className="mt-2 max-w-2xl text-xs leading-5 text-muted-foreground">
                Current English-language coverage from configured providers, scored by the trained
                Trading Wens financial sentiment model.
              </p>
            </div>
          </div>

          {newsQuery.error ? (
            <ProviderError
              message={
                newsQuery.error instanceof Error
                  ? newsQuery.error.message
                  : "Company news is unavailable."
              }
              onRetry={() => void newsQuery.refetch()}
            />
          ) : (
            <div className="news-grid mt-6">
              {newsQuery.isPending
                ? Array.from({ length: 4 }, (_, index) => (
                    <div className="news-card" key={`news-loading-${index}`}>
                      <div className="h-3 w-28 animate-pulse rounded bg-secondary" />
                      <div className="mt-4 h-4 animate-pulse rounded bg-secondary" />
                      <div className="mt-2 h-4 w-3/4 animate-pulse rounded bg-secondary" />
                    </div>
                  ))
                : newsQuery.data?.data.map((article) => (
                    <article className="news-card" key={article.id}>
                      <div className="flex flex-wrap items-center gap-2 text-[10px] text-muted-foreground">
                        <span className="uppercase tracking-wider">
                          {article.provider} · {article.domain}
                        </span>
                        <span aria-hidden="true">·</span>
                        <time dateTime={article.published_at}>
                          {formatDistanceToNowStrict(new Date(article.published_at), {
                            addSuffix: true,
                          })}
                        </time>
                      </div>
                      <h3 className="mt-3 text-sm font-medium leading-6">{article.title}</h3>
                      <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
                        <SentimentBadge sentiment={article.sentiment} />
                        <a
                          href={article.url}
                          target="_blank"
                          rel="noreferrer noopener"
                          className="inline-flex min-h-11 items-center gap-1.5 text-xs text-primary hover:underline"
                          aria-label={`Read ${article.title}`}
                        >
                          Read source <ExternalLink size={13} />
                        </a>
                      </div>
                    </article>
                  ))}
              {!newsQuery.isPending && newsQuery.data?.data.length === 0 && (
                <div className="col-span-full py-12 text-center text-sm text-muted-foreground">
                  No recent coverage was returned for {symbol}.
                </div>
              )}
            </div>
          )}

          <div className="mt-5 flex flex-wrap justify-between gap-2 text-[11px] text-muted-foreground">
            <span>
              {newsQuery.data
                ? `${newsQuery.data.count} focused articles · ${newsQuery.data.providers.join(" + ")}`
                : "Connecting to news providers"}
            </span>
            <span>
              {newsQuery.data
                ? `Feed checked ${new Date(newsQuery.data.as_of).toLocaleString()}`
                : "Waiting for company news"}
            </span>
          </div>
        </section>

        <footer className="mt-7 flex flex-wrap items-center justify-between gap-3 pb-7 text-[11px] text-muted-foreground">
          <span>PRICE DATA BY ALPACA · NEWS BY ALPACA + GDELT · NLP BY TRADING WENS</span>
          <span>Market data may be delayed by feed entitlement. Not financial advice.</span>
        </footer>
      </main>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric-card">
      <div className="label text-muted-foreground">{label}</div>
      <div className="mt-3 text-lg font-semibold tabular-nums">{value}</div>
    </div>
  );
}

function ProviderError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="mt-7 flex min-h-52 flex-col items-center justify-center rounded-md border border-destructive/30 bg-destructive/5 px-6 text-center">
      <WifiOff size={24} className="text-destructive" />
      <p role="alert" className="mt-3 text-sm font-medium">
        {message}
      </p>
      <Button variant="outline" size="sm" className="mt-4" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}
