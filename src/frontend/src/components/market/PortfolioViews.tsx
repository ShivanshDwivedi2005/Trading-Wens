import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { formatDistanceToNowStrict } from "date-fns";
import {
  ArrowDownRight,
  ArrowRight,
  ArrowUpRight,
  ExternalLink,
  Minus,
  RefreshCw,
  ShieldCheck,
  WifiOff,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { fetchMarketSnapshots, type MarketSnapshot } from "@/lib/market-api";
import { priceMovementPrediction, type PriceMovementPrediction } from "@/lib/market-prediction";
import { fetchMarketNews } from "@/lib/news-api";
import { portfolioRisk, positionRisk, sentimentWatchlist } from "@/lib/portfolio-risk";
import { fetchTradingPortfolio } from "@/lib/trading-api";
import { AccountExecutionOverview } from "./AccountExecutionOverview";

const money = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 2,
});

const compactMoney = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  notation: "compact",
  maximumFractionDigits: 1,
});

function usePortfolioQuery() {
  return useQuery({
    queryKey: ["trading", "portfolio"],
    queryFn: fetchTradingPortfolio,
    staleTime: 5_000,
    refetchInterval: 15_000,
    retry: 1,
  });
}

function useSnapshotsQuery() {
  return useQuery({
    queryKey: ["market", "sp500-top-30"],
    queryFn: fetchMarketSnapshots,
    staleTime: 10_000,
    refetchInterval: 15_000,
    retry: 1,
  });
}

function useNewsQuery() {
  return useQuery({
    queryKey: ["news", "large-cap-market"],
    queryFn: () => fetchMarketNews(),
    staleTime: 30_000,
    refetchInterval: 30_000,
    retry: 1,
  });
}

export function AccountOverview() {
  const portfolio = usePortfolioQuery();
  const snapshots = useSnapshotsQuery();
  const news = useNewsQuery();
  if (portfolio.error) {
    return <DataError error={portfolio.error} onRetry={() => void portfolio.refetch()} />;
  }
  if (portfolio.isPending) return <DashboardSkeleton />;

  const positions = portfolio.data.positions;
  const invested = new Set(positions.map((position) => position.symbol));
  const positionStocks = positions.map((position) => {
    const snapshot = snapshots.data?.data.find((item) => item.symbol === position.symbol);
    return {
      symbol: position.symbol,
      name: snapshot?.name ?? "Invested position",
      price: position.current_price,
      change: position.change_today,
      invested: true,
    };
  });
  const importantStocks = (snapshots.data?.data ?? [])
    .filter((snapshot) => !invested.has(snapshot.symbol))
    .slice(0, Math.max(0, 8 - positionStocks.length))
    .map((snapshot) => ({
      symbol: snapshot.symbol,
      name: snapshot.name,
      price: snapshot.price,
      change: snapshot.change_percent,
      invested: false,
    }));
  const visibleStocks = [...positionStocks, ...importantStocks].slice(0, 8);
  const watchlist = sentimentWatchlist(snapshots.data?.data ?? [], news.data?.data ?? []);
  const accountNews = (news.data?.data ?? [])
    .filter((article) => article.matched_symbols.some((symbol) => invested.has(symbol)))
    .slice(0, 5);
  const dailyChange = portfolio.data.account.equity - portfolio.data.account.last_equity;

  return (
    <div className="space-y-5">
      <section className="metric-grid mt-0">
        <Metric
          label="PORTFOLIO VALUE"
          value={money.format(portfolio.data.account.portfolio_value)}
        />
        <Metric
          label="TODAY'S P/L"
          value={`${dailyChange >= 0 ? "+" : ""}${money.format(dailyChange)}`}
          tone={dailyChange >= 0 ? "positive" : "negative"}
        />
        <Metric label="BUYING POWER" value={money.format(portfolio.data.account.buying_power)} />
        <Metric label="OPEN POSITIONS" value={String(positions.length)} />
      </section>

      <AccountExecutionOverview portfolio={portfolio.data} />

      <section className="panel">
        <SectionHeading
          eyebrow="ACCOUNT + MARKET WATCH"
          title="Your eight-stock view"
          description="Open Alpaca positions first, completed with leading tracked stocks."
        />
        <div className="overview-stock-grid mt-6">
          {visibleStocks.map((stock) => (
            <div className="overview-stock" key={stock.symbol}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="text-sm font-semibold">{stock.symbol}</div>
                  <div className="mt-1 truncate text-[10px] text-muted-foreground">
                    {stock.name}
                  </div>
                </div>
                <span className={stock.invested ? "position-tag" : "watch-tag"}>
                  {stock.invested ? "Invested" : "Watch"}
                </span>
              </div>
              <div className="mt-4 text-lg font-semibold tabular-nums">
                {money.format(stock.price)}
              </div>
              <div className={`mt-1 text-xs ${stock.change >= 0 ? "text-profit" : "text-loss"}`}>
                {stock.change >= 0 ? "+" : ""}
                {stock.change.toFixed(2)}% today
              </div>
            </div>
          ))}
          {!visibleStocks.length && <EmptyState text="No stock data is available yet." />}
        </div>
      </section>

      <div className="overview-columns">
        <section className="panel">
          <SectionHeading
            eyebrow="TRAINED FINBERT SENTIMENT"
            title="Top five sentiment watchlist"
            description="Ranked from real news headlines; research aid only, not an investment recommendation."
          />
          <AnalysisStatus feed={news.data} onRetry={() => void news.refetch()} />
          <div className="mt-5 space-y-1">
            {watchlist.map((item, index) => (
              <div className="sentiment-rank" key={item.snapshot.symbol}>
                <span className="rank-number">{index + 1}</span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <strong className="text-xs">{item.snapshot.symbol}</strong>
                    <span className="truncate text-[10px] text-muted-foreground">
                      {item.snapshot.name}
                    </span>
                  </div>
                  <div className="mt-1 text-[10px] text-muted-foreground">
                    {item.analyzedCount} of {item.articleCount} headlines analyzed ·{" "}
                    {item.confidence}% model confidence
                  </div>
                </div>
                {item.score === null ? (
                  <span className="text-[10px] text-muted-foreground">Unavailable</span>
                ) : (
                  <span className={item.score >= 0 ? "text-profit" : "text-loss"}>
                    {item.score >= 0 ? "+" : ""}
                    {item.score.toFixed(2)}
                  </span>
                )}
              </div>
            ))}
          </div>
        </section>

        <section className="panel">
          <SectionHeading
            eyebrow="ACCOUNT-RELEVANT COVERAGE"
            title="Latest position news"
            description="The five newest Alpaca and GDELT headlines matched to current holdings."
          />
          <div className="mt-5 divide-y divide-border">
            {accountNews.map((article) => (
              <a
                key={article.id}
                href={article.url}
                target="_blank"
                rel="noreferrer noopener"
                className="group flex min-h-16 items-center gap-3 py-3"
              >
                <div className="min-w-0 flex-1">
                  <div className="line-clamp-2 text-xs leading-5 group-hover:text-primary">
                    {article.title}
                  </div>
                  <div className="mt-1 text-[10px] text-muted-foreground">
                    {article.domain} ·{" "}
                    {formatDistanceToNowStrict(new Date(article.published_at), { addSuffix: true })}
                  </div>
                </div>
                <ExternalLink size={13} className="shrink-0 text-muted-foreground" />
              </a>
            ))}
            {!accountNews.length && (
              <EmptyState text="No recent headlines matched current positions." />
            )}
          </div>
        </section>
      </div>
      <ProviderStamp asOf={portfolio.data.as_of} />
    </div>
  );
}

export function LiveRiskFeed() {
  const portfolio = usePortfolioQuery();
  const snapshots = useSnapshotsQuery();
  const news = useNewsQuery();
  if (snapshots.error)
    return <DataError error={snapshots.error} onRetry={() => void snapshots.refetch()} />;
  if (snapshots.isPending) return <DashboardSkeleton />;
  const positions = portfolio.data?.positions ?? [];
  const positionsBySymbol = new Map(positions.map((position) => [position.symbol, position]));

  return (
    <section className="panel">
      <SectionHeading
        eyebrow="TOP 30 LIVE MARKET RISK"
        title="Large-cap risk feed"
        description="Every maintained top-30 stock stays visible. Live price action and trained-model news sentiment produce a directional movement prediction; open Alpaca positions are overlaid when available."
      />
      <AnalysisStatus feed={news.data} onRetry={() => void news.refetch()} />
      {portfolio.error && (
        <div
          role="status"
          className="mt-5 flex flex-wrap items-center justify-between gap-3 rounded border border-accent/30 bg-accent/5 px-4 py-3 text-xs"
        >
          <span>Live market risk is available, but the position overlay could not be loaded.</span>
          <Button variant="outline" size="sm" onClick={() => void portfolio.refetch()}>
            <RefreshCw size={13} /> Retry positions
          </Button>
        </div>
      )}
      <div className="mt-6 overflow-x-auto">
        <div className="risk-table min-w-[1080px]">
          <div className="risk-row market-risk-row risk-head">
            <span>Company</span>
            <span>Last</span>
            <span>Today</span>
            <span>Day range</span>
            <span>Volume</span>
            <span>Prediction / risk</span>
            <span>Position / analysis</span>
          </div>
          {snapshots.data.data.map((snapshot) => {
            const risk = marketSnapshotRisk(snapshot);
            const prediction = priceMovementPrediction(snapshot, news.data?.data ?? []);
            const position = positionsBySymbol.get(snapshot.symbol);
            return (
              <div className="risk-row market-risk-row risk-entry" key={snapshot.symbol}>
                <span>
                  <strong>{snapshot.symbol}</strong>
                  <small>{snapshot.name}</small>
                </span>
                <span className="tabular-nums">
                  {snapshot.available ? money.format(snapshot.price) : "—"}
                </span>
                <span
                  className={`tabular-nums ${snapshot.change_percent >= 0 ? "text-profit" : "text-loss"}`}
                >
                  {snapshot.available ? (
                    <>
                      {snapshot.change_percent >= 0 ? "+" : ""}
                      {snapshot.change_percent.toFixed(2)}%
                    </>
                  ) : (
                    "Unavailable"
                  )}
                </span>
                <span className="tabular-nums">
                  {snapshot.available
                    ? `${money.format(snapshot.low)} – ${money.format(snapshot.high)}`
                    : "—"}
                </span>
                <span className="tabular-nums">{formatVolume(snapshot.volume)}</span>
                <span>
                  <MovementBadge prediction={prediction} />
                  <span className={`risk-badge risk-${risk.level.toLowerCase()}`}>
                    {risk.level} · {risk.score.toFixed(1)}
                  </span>
                </span>
                <span className="flex items-center justify-between gap-2">
                  <span className={position ? "position-tag" : "watch-tag"}>
                    {position ? `${position.quantity.toLocaleString()} shares` : "Watch"}
                  </span>
                  <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="min-h-11 justify-start px-2 text-xs"
                  >
                    <Link to="/stocks/$symbol" params={{ symbol: snapshot.symbol }}>
                      <ArrowRight size={13} className="text-primary" />
                      Analyze
                    </Link>
                  </Button>
                </span>
              </div>
            );
          })}
          {!snapshots.data.data.length && (
            <EmptyState text="No tracked market snapshots are available yet." />
          )}
        </div>
      </div>
      <ProviderStamp asOf={snapshots.data.as_of} />
    </section>
  );
}

function marketSnapshotRisk(snapshot: MarketSnapshot) {
  if (!snapshot.available || snapshot.price <= 0) {
    return { level: "Low" as const, score: 0 };
  }

  const intradayRange = ((snapshot.high - snapshot.low) / snapshot.price) * 100;
  const score = Math.min(10, Math.abs(snapshot.change_percent) * 1.4 + intradayRange * 0.8);
  const level = score >= 7 ? "High" : score >= 4 ? "Moderate" : "Low";

  return { level, score };
}

function formatVolume(value: number) {
  return new Intl.NumberFormat("en-US", {
    notation: "compact",
    maximumFractionDigits: 1,
  }).format(value);
}

export function PortfolioStatus() {
  const portfolio = usePortfolioQuery();
  const snapshots = useSnapshotsQuery();
  const news = useNewsQuery();
  if (portfolio.error)
    return <DataError error={portfolio.error} onRetry={() => void portfolio.refetch()} />;
  if (portfolio.isPending) return <DashboardSkeleton />;
  const account = portfolio.data.account;
  const predictions = new Map(
    portfolio.data.positions.map((position) => {
      const snapshot = snapshotForPosition(position, snapshots.data?.data ?? []);
      return [position.symbol, priceMovementPrediction(snapshot, news.data?.data ?? [])] as const;
    }),
  );
  const overall = portfolioRisk(portfolio.data.positions, account.portfolio_value, predictions);
  const dailyChange = account.equity - account.last_equity;
  return (
    <div className="space-y-5">
      <section className="portfolio-risk-hero">
        <div>
          <div className="label text-muted-foreground">OVERALL PORTFOLIO RISK</div>
          <div className="mt-3 flex items-baseline gap-3">
            <span className="text-5xl font-semibold tabular-nums">{overall.score.toFixed(1)}</span>
            <span className="text-sm text-muted-foreground">/ 10 · {overall.level}</span>
          </div>
          <p className="mt-3 max-w-xl text-xs leading-5 text-muted-foreground">
            Cumulative position-weighted risk from live exposure and adverse price-movement
            predictions. Predictions are research signals, not guaranteed outcomes.
          </p>
        </div>
        <ShieldCheck
          size={38}
          className={overall.level === "High" ? "text-destructive" : "text-primary"}
        />
      </section>
      <section className="metric-grid mt-0">
        <Metric label="LARGEST POSITION" value={`${overall.concentration.toFixed(1)}%`} />
        <Metric label="ADVERSE POSITION RISK" value={`${overall.downside.toFixed(1)} / 10`} />
        <Metric
          label="PREDICTION ALIGNMENT"
          value={`${overall.alignedCount} aligned · ${overall.adverseCount} at risk`}
          tone={overall.adverseCount > 0 ? "negative" : "positive"}
        />
        <Metric
          label="TODAY'S EQUITY CHANGE"
          value={money.format(dailyChange)}
          tone={dailyChange >= 0 ? "positive" : "negative"}
        />
      </section>
      <section className="panel">
        <SectionHeading
          eyebrow="INDIVIDUAL EXPOSURE"
          title="Position-level risk"
          description="Every open position is evaluated against its predicted move. A move against the position is marked as risk; an aligned move is marked green."
        />
        <AnalysisStatus feed={news.data} onRetry={() => void news.refetch()} />
        <div className="portfolio-position-grid mt-6">
          {portfolio.data.positions.map((position) => {
            const prediction = predictions.get(position.symbol);
            const risk = positionRisk(position, account.portfolio_value, prediction);
            return (
              <article
                className={`position-risk-card position-risk-${risk.alignment}`}
                key={position.symbol}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h3 className="text-base font-semibold">{position.symbol}</h3>
                    <p className="mt-1 text-[10px] text-muted-foreground">
                      {position.quantity.toLocaleString()} shares · {position.side}
                    </p>
                  </div>
                  <div className="flex flex-wrap justify-end gap-2">
                    {prediction && <MovementBadge prediction={prediction} />}
                    <span className={`risk-badge risk-${risk.level.toLowerCase()}`}>
                      {risk.alignment === "adverse"
                        ? "At risk"
                        : risk.alignment === "aligned"
                          ? "Aligned"
                          : risk.level}
                    </span>
                  </div>
                </div>
                <div className="mt-5 grid grid-cols-2 gap-4 text-xs">
                  <Value label="Risk score" value={`${risk.score.toFixed(1)} / 10`} />
                  <Value label="Portfolio weight" value={`${risk.weight.toFixed(1)}%`} />
                  <Value
                    label="Move possibility"
                    value={prediction ? `${prediction.possibility}%` : "Unavailable"}
                  />
                  <Value label="Market value" value={compactMoney.format(position.market_value)} />
                  <Value
                    label="Unrealized P/L"
                    value={money.format(position.unrealized_pl)}
                    tone={position.unrealized_pl >= 0 ? "positive" : "negative"}
                  />
                </div>
                <div className="mt-5 border-t border-border pt-4 text-xs">
                  <span className="text-muted-foreground">Action: </span>
                  {risk.action}
                </div>
              </article>
            );
          })}
          {!portfolio.data.positions.length && <EmptyState text="No open positions to assess." />}
        </div>
      </section>
      <ProviderStamp asOf={portfolio.data.as_of} />
    </div>
  );
}

function snapshotForPosition(
  position: Parameters<typeof positionRisk>[0],
  snapshots: MarketSnapshot[],
): MarketSnapshot {
  return (
    snapshots.find((snapshot) => snapshot.symbol === position.symbol) ?? {
      symbol: position.symbol,
      name: position.symbol,
      price: position.current_price,
      change: 0,
      change_percent: position.change_today,
      open: position.current_price,
      high: position.current_price,
      low: position.current_price,
      previous_close: position.current_price,
      volume: 0,
      timestamp: "",
      available: position.current_price > 0,
    }
  );
}

function MovementBadge({ prediction }: { prediction: PriceMovementPrediction }) {
  const Icon =
    prediction.direction === "up"
      ? ArrowUpRight
      : prediction.direction === "down"
        ? ArrowDownRight
        : Minus;
  return (
    <span className={`movement-badge movement-${prediction.direction}`}>
      <Icon size={12} aria-hidden="true" />
      {prediction.label} · {prediction.possibility}%
    </span>
  );
}

function AnalysisStatus({
  feed,
  onRetry,
}: {
  feed: Awaited<ReturnType<typeof fetchMarketNews>> | undefined;
  onRetry: () => void;
}) {
  if (!feed || feed.analysis_status !== "unavailable") return null;
  return (
    <div className="analysis-status mt-5" role="status">
      <span>
        Headlines loaded, but trained-model analysis is unavailable. Start the inference service on
        port 8090, then retry.
      </span>
      <Button variant="outline" size="sm" onClick={onRetry}>
        <RefreshCw size={13} /> Retry analysis
      </Button>
    </div>
  );
}

function Metric({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "positive" | "negative";
}) {
  return (
    <div className="metric">
      <div className="label text-muted-foreground">{label}</div>
      <div
        className={`mt-5 text-2xl font-semibold tabular-nums ${tone === "positive" ? "text-profit" : tone === "negative" ? "text-loss" : ""}`}
      >
        {value}
      </div>
    </div>
  );
}

function Value({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "positive" | "negative";
}) {
  return (
    <div>
      <div className="text-[10px] text-muted-foreground">{label}</div>
      <div
        className={`mt-1 font-medium tabular-nums ${tone === "positive" ? "text-profit" : tone === "negative" ? "text-loss" : ""}`}
      >
        {value}
      </div>
    </div>
  );
}

function SectionHeading({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string;
  title: string;
  description: string;
}) {
  return (
    <div>
      <div className="label text-primary">{eyebrow}</div>
      <h2 className="mt-2 text-lg font-semibold">{title}</h2>
      <p className="mt-2 max-w-2xl text-xs leading-5 text-muted-foreground">{description}</p>
    </div>
  );
}

function ProviderStamp({ asOf }: { asOf: string }) {
  return (
    <div className="mt-5 flex flex-wrap justify-between gap-2 text-[11px] text-muted-foreground">
      <span>ALPACA SIMULATION ACCOUNT · LIVE PROVIDER DATA</span>
      <span>Updated {new Date(asOf).toLocaleString()}</span>
    </div>
  );
}

function EmptyState({ text }: { text: string }) {
  return <div className="py-8 text-center text-xs text-muted-foreground">{text}</div>;
}

function DashboardSkeleton() {
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="h-44 animate-pulse rounded-md bg-secondary" />
      <div className="h-44 animate-pulse rounded-md bg-secondary" />
      <div className="h-72 animate-pulse rounded-md bg-secondary sm:col-span-2" />
    </div>
  );
}

function DataError({ error, onRetry }: { error: Error; onRetry: () => void }) {
  return (
    <div className="panel flex min-h-72 flex-col items-center justify-center text-center">
      <WifiOff className="text-destructive" />
      <p role="alert" className="mt-3 text-sm">
        {error.message}
      </p>
      <Button variant="outline" className="mt-4 min-h-11" onClick={onRetry}>
        <RefreshCw size={14} /> Retry
      </Button>
    </div>
  );
}
