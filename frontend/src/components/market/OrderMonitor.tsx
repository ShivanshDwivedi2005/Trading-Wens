import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { format } from "date-fns";
import {
  Activity,
  ArrowLeft,
  CheckCircle2,
  Clock3,
  FileClock,
  RefreshCw,
  Search,
  ShieldCheck,
  TrendingDown,
  TrendingUp,
  WifiOff,
  XCircle,
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
import { fetchMarketNews } from "@/lib/news-api";
import { fillMarkDelta, positionOutlook } from "@/lib/trading-analysis";
import {
  cancelPaperOrder,
  fetchOrderMonitor,
  fetchTradingPortfolio,
  type OrderAuditEvent,
  type PaperFill,
  type PaperOrder,
} from "@/lib/trading-api";

type MonitorView = "orders" | "fills" | "audit";

const money = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  maximumFractionDigits: 2,
});

export function OrderMonitor({ initialSymbol = "" }: { initialSymbol?: string }) {
  const queryClient = useQueryClient();
  const [view, setView] = useState<MonitorView>("orders");
  const [symbolFilter, setSymbolFilter] = useState(initialSymbol.toUpperCase());
  const [cancelTarget, setCancelTarget] = useState<PaperOrder | null>(null);
  const monitor = useQuery({
    queryKey: ["trading", "orders"],
    queryFn: fetchOrderMonitor,
    staleTime: 5_000,
    refetchInterval: 10_000,
    retry: 1,
  });
  const portfolio = useQuery({
    queryKey: ["trading", "portfolio"],
    queryFn: fetchTradingPortfolio,
    staleTime: 5_000,
    refetchInterval: 10_000,
    retry: 1,
  });
  const news = useQuery({
    queryKey: ["news", "large-cap-market"],
    queryFn: () => fetchMarketNews(),
    staleTime: 120_000,
    retry: 1,
  });
  const cancelOrder = useMutation({
    mutationFn: cancelPaperOrder,
    onSuccess: async () => {
      setCancelTarget(null);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ["trading", "orders"] }),
        queryClient.invalidateQueries({ queryKey: ["trading", "portfolio"] }),
      ]);
    },
  });

  const normalizedFilter = symbolFilter.trim().toUpperCase();
  const visibleOrders = useMemo(
    () =>
      (monitor.data?.orders ?? []).filter(
        (order) => !normalizedFilter || order.symbol.includes(normalizedFilter),
      ),
    [monitor.data, normalizedFilter],
  );
  const visibleFills = useMemo(
    () =>
      (monitor.data?.fills ?? []).filter(
        (fill) => !normalizedFilter || fill.symbol.includes(normalizedFilter),
      ),
    [monitor.data, normalizedFilter],
  );
  const visibleAudit = useMemo(
    () =>
      (monitor.data?.audit_trail ?? []).filter(
        (event) => !normalizedFilter || event.symbol.includes(normalizedFilter),
      ),
    [monitor.data, normalizedFilter],
  );
  const focusPosition = normalizedFilter
    ? portfolio.data?.positions.find((position) => position.symbol === normalizedFilter)
    : portfolio.data?.positions[0];
  const outlook = positionOutlook(focusPosition, news.data?.data ?? []);
  const refreshing = monitor.isFetching || portfolio.isFetching || news.isFetching;
  const hasError = monitor.error || portfolio.error;

  const refresh = () => {
    void Promise.all([monitor.refetch(), portfolio.refetch(), news.refetch()]);
  };

  return (
    <div className="order-monitor min-h-screen bg-background text-foreground">
      <header className="terminal-header">
        <div className="flex min-w-0 items-center gap-3">
          <Button asChild variant="ghost" size="icon" aria-label="Back to workspace">
            <Link to="/dashboard">
              <ArrowLeft size={17} />
            </Link>
          </Button>
          <span className="brand-mark hidden sm:flex">
            <Activity size={16} />
          </span>
          <div className="min-w-0">
            <div className="label text-primary">ORDER MANAGEMENT</div>
            <h1 className="truncate text-base font-semibold sm:text-lg">Execution monitor</h1>
          </div>
        </div>
        <div className="flex items-center gap-2 sm:gap-4">
          <span className="paper-notice hidden sm:block">ALPACA PAPER</span>
          <span className="hidden text-[10px] text-muted-foreground md:block">
            {monitor.data
              ? `Updated ${new Date(monitor.data.as_of).toLocaleTimeString()}`
              : "Syncing"}
          </span>
          <Button
            variant="outline"
            size="icon"
            className="size-11"
            aria-label="Refresh orders, fills, and position evidence"
            onClick={refresh}
            disabled={refreshing}
          >
            <RefreshCw size={15} className={refreshing ? "animate-spin" : ""} />
          </Button>
        </div>
      </header>

      <main className="terminal-content">
        <section className="terminal-summary-grid">
          <TerminalMetric
            label="WORKING ORDERS"
            value={String(monitor.data?.working_count ?? 0)}
            detail={`${monitor.data?.order_count ?? 0} total order records`}
          />
          <TerminalMetric
            label="FILL EVENTS"
            value={String(monitor.data?.fill_count ?? 0)}
            detail="Latest 100 Alpaca activities"
          />
          <TerminalMetric
            label="OPEN POSITIONS"
            value={String(portfolio.data?.positions.length ?? 0)}
            detail={
              portfolio.data
                ? money.format(portfolio.data.account.portfolio_value)
                : "Loading equity"
            }
          />
          <TerminalMetric
            label="ACCOUNT"
            value={portfolio.data?.account.status ?? "—"}
            detail={
              portfolio.data ? `${portfolio.data.account.id.slice(0, 8)}… · paper` : "Verifying"
            }
          />
        </section>

        {hasError ? (
          <section className="terminal-error" role="alert">
            <WifiOff size={22} />
            <div>
              <strong>Order activity is unavailable.</strong>
              <p>Check the Alpaca paper connection, then retry.</p>
            </div>
            <Button variant="outline" onClick={refresh}>
              Retry
            </Button>
          </section>
        ) : (
          <div className="terminal-layout">
            <section className="terminal-panel min-w-0">
              <div className="terminal-toolbar">
                <div className="terminal-tabs" role="tablist" aria-label="Order activity views">
                  <TerminalTab
                    active={view === "orders"}
                    label="Order Book"
                    count={visibleOrders.length}
                    icon={Clock3}
                    onClick={() => setView("orders")}
                  />
                  <TerminalTab
                    active={view === "fills"}
                    label="Fills"
                    count={visibleFills.length}
                    icon={CheckCircle2}
                    onClick={() => setView("fills")}
                  />
                  <TerminalTab
                    active={view === "audit"}
                    label="Audit Trail"
                    count={visibleAudit.length}
                    icon={FileClock}
                    onClick={() => setView("audit")}
                  />
                </div>
                <div className="relative w-full sm:w-52">
                  <Search size={14} className="absolute left-3 top-3.5 text-muted-foreground" />
                  <Input
                    aria-label="Filter order activity by symbol"
                    placeholder="Filter symbol"
                    value={symbolFilter}
                    onChange={(event) => setSymbolFilter(event.target.value.toUpperCase())}
                    className="h-11 pl-9 font-mono text-xs uppercase"
                  />
                </div>
              </div>

              {monitor.isPending ? (
                <div className="terminal-loading" aria-label="Loading order activity">
                  <div className="h-10 animate-pulse rounded bg-secondary" />
                  <div className="h-10 animate-pulse rounded bg-secondary" />
                  <div className="h-10 animate-pulse rounded bg-secondary" />
                </div>
              ) : view === "orders" ? (
                <OrderBook orders={visibleOrders} onCancel={setCancelTarget} />
              ) : view === "fills" ? (
                <FillsView fills={visibleFills} positions={portfolio.data?.positions ?? []} />
              ) : (
                <AuditTrail events={visibleAudit} />
              )}
            </section>

            <aside className="outlook-panel">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="label text-muted-foreground">POSITION OUTLOOK</div>
                  <h2 className="mt-2 text-lg font-semibold">
                    {focusPosition?.symbol ?? "No open position"}
                  </h2>
                </div>
                <ShieldCheck size={21} className="text-primary" />
              </div>
              <div className={`outlook-score outlook-${outlook.direction}`}>
                {outlook.direction === "loss" ? (
                  <TrendingDown size={19} />
                ) : (
                  <TrendingUp size={19} />
                )}
                <div>
                  <span>
                    {outlook.direction === "balanced"
                      ? "Balanced"
                      : `${capitalize(outlook.direction)} possibility`}
                  </span>
                  <strong>{outlook.possibility}%</strong>
                </div>
              </div>
              <div className="outlook-metrics">
                <div>
                  <span>Score</span>
                  <strong>{outlook.score.toFixed(1)} / 10</strong>
                </div>
                <div>
                  <span>Confidence</span>
                  <strong>{outlook.confidence}%</strong>
                </div>
              </div>
              <div className="mt-5">
                <div className="label text-muted-foreground">WHY THIS RESULT</div>
                <ul className="outlook-reasons">
                  {outlook.reasons.map((reason) => (
                    <li key={reason}>{reason}</li>
                  ))}
                </ul>
              </div>
              <div className="mt-5 border-t border-border pt-4">
                <div className="label text-muted-foreground">EVIDENCE SOURCES</div>
                <div className="mt-3 space-y-2">
                  {outlook.sources.map((source) => (
                    <div className="source-row" key={source}>
                      <CheckCircle2 size={12} aria-hidden="true" /> {source}
                    </div>
                  ))}
                </div>
              </div>
              <p className="mt-5 text-[10px] leading-4 text-muted-foreground">
                Directional scenario, not a guaranteed probability or investment recommendation.
                Scores combine current marks, account P/L, and available model sentiment.
              </p>
            </aside>
          </div>
        )}
      </main>

      <AlertDialog
        open={Boolean(cancelTarget)}
        onOpenChange={(open) => !open && setCancelTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cancel working paper order?</AlertDialogTitle>
            <AlertDialogDescription>
              This sends a cancellation request for {cancelTarget?.symbol} order{" "}
              {cancelTarget?.id.slice(0, 8)}… to Alpaca. Already filled quantity cannot be canceled.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {cancelOrder.error && (
            <p role="alert" className="text-sm text-destructive">
              {cancelOrder.error.message}
            </p>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={cancelOrder.isPending}>Keep order</AlertDialogCancel>
            <AlertDialogAction
              disabled={cancelOrder.isPending || !cancelTarget}
              onClick={(event) => {
                event.preventDefault();
                if (cancelTarget) cancelOrder.mutate(cancelTarget.id);
              }}
            >
              {cancelOrder.isPending ? "Canceling…" : "Cancel paper order"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function TerminalMetric({
  label,
  value,
  detail,
}: {
  label: string;
  value: string;
  detail: string;
}) {
  return (
    <div className="terminal-metric">
      <div className="label text-muted-foreground">{label}</div>
      <div className="mt-2 text-xl font-semibold tabular-nums">{value}</div>
      <div className="mt-1 text-[10px] text-muted-foreground">{detail}</div>
    </div>
  );
}

function TerminalTab({
  active,
  label,
  count,
  icon: Icon,
  onClick,
}: {
  active: boolean;
  label: string;
  count: number;
  icon: typeof Clock3;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      className={`terminal-tab ${active ? "terminal-tab-active" : ""}`}
      onClick={onClick}
    >
      <Icon size={14} /> {label} <span>{count}</span>
    </button>
  );
}

function OrderBook({
  orders,
  onCancel,
}: {
  orders: PaperOrder[];
  onCancel: (order: PaperOrder) => void;
}) {
  return (
    <div className="overflow-x-auto">
      <div className="order-grid min-w-[980px]" role="table" aria-label="Alpaca paper order book">
        <div className="order-grid-row order-grid-head" role="row">
          <span>Time</span>
          <span>Contract</span>
          <span>B/S</span>
          <span>Type</span>
          <span>Quantity</span>
          <span>Executed</span>
          <span>Price</span>
          <span>Status</span>
          <span>Action</span>
        </div>
        {orders.map((order) => (
          <div className="order-grid-row order-grid-entry" role="row" key={order.id}>
            <time dateTime={order.submitted_at}>
              {format(new Date(order.submitted_at), "MMM d HH:mm:ss")}
            </time>
            <strong>{order.symbol}</strong>
            <SideBadge side={order.side} />
            <span className="uppercase">
              {order.type} · {order.time_in_force}
            </span>
            <span className="tabular-nums">{order.quantity}</span>
            <span className="tabular-nums">{order.filled_quantity}</span>
            <span className="tabular-nums">
              {order.limit_price ? money.format(order.limit_price) : "MKT"}
            </span>
            <StatusBadge status={order.status} working={order.working} />
            {order.working ? (
              <Button
                variant="ghost"
                size="sm"
                className="min-h-11 text-destructive"
                onClick={() => onCancel(order)}
              >
                <XCircle size={13} /> Cancel
              </Button>
            ) : (
              <span className="text-[10px] text-muted-foreground">Complete</span>
            )}
          </div>
        ))}
        {!orders.length && <EmptyRow text="No order records match this symbol filter." />}
      </div>
    </div>
  );
}

function FillsView({
  fills,
  positions,
}: {
  fills: PaperFill[];
  positions: Array<{ symbol: string; current_price: number }>;
}) {
  const marks = new Map(positions.map((position) => [position.symbol, position.current_price]));
  return (
    <div className="overflow-x-auto">
      <div className="fill-grid min-w-[860px]" role="table" aria-label="Alpaca fill activity">
        <div className="fill-grid-row order-grid-head" role="row">
          <span>Execution time</span>
          <span>Contract</span>
          <span>B/S</span>
          <span>Quantity</span>
          <span>Fill price</span>
          <span>Current mark</span>
          <span>Mark delta</span>
          <span>Order</span>
        </div>
        {fills.map((fill) => {
          const mark = marks.get(fill.symbol);
          const delta = fillMarkDelta(fill, mark);
          return (
            <div className="fill-grid-row order-grid-entry" role="row" key={fill.id}>
              <time dateTime={fill.transaction_time}>
                {format(new Date(fill.transaction_time), "MMM d HH:mm:ss.SSS")}
              </time>
              <strong>{fill.symbol}</strong>
              <SideBadge side={fill.side} />
              <span className="tabular-nums">{fill.quantity}</span>
              <span className="tabular-nums">{money.format(fill.price)}</span>
              <span className="tabular-nums">{mark ? money.format(mark) : "—"}</span>
              <span
                className={`tabular-nums ${delta === null ? "text-muted-foreground" : delta >= 0 ? "outlook-profit" : "outlook-loss"}`}
              >
                {delta === null ? "No open mark" : `${delta >= 0 ? "+" : ""}${money.format(delta)}`}
              </span>
              <span className="font-mono text-[10px]">{fill.order_id.slice(0, 8)}…</span>
            </div>
          );
        })}
        {!fills.length && <EmptyRow text="No fill activity matches this symbol filter." />}
      </div>
    </div>
  );
}

function AuditTrail({ events }: { events: OrderAuditEvent[] }) {
  return (
    <div className="overflow-x-auto">
      <div className="audit-grid min-w-[1080px]" role="table" aria-label="Paper order audit trail">
        <div className="audit-grid-row order-grid-head" role="row">
          <span>Timestamp</span>
          <span>Event</span>
          <span>Contract</span>
          <span>B/S</span>
          <span>Qty / Exe</span>
          <span>Price</span>
          <span>Message</span>
          <span>Source</span>
        </div>
        {events.map((event) => (
          <div className="audit-grid-row order-grid-entry" role="row" key={event.id}>
            <time dateTime={event.timestamp}>
              {format(new Date(event.timestamp), "MMM d HH:mm:ss.SSS")}
            </time>
            <StatusBadge status={event.event} working={event.event === "STATUS"} />
            <strong>{event.symbol}</strong>
            <SideBadge side={event.side} />
            <span className="tabular-nums">
              {event.quantity} / {event.filled_quantity}
            </span>
            <span className="tabular-nums">{event.price ? money.format(event.price) : "—"}</span>
            <span>{event.message}</span>
            <span className="font-mono text-[9px] uppercase text-muted-foreground">
              {event.source.replaceAll("_", " ")}
            </span>
          </div>
        ))}
        {!events.length && <EmptyRow text="No audit messages match this symbol filter." />}
      </div>
    </div>
  );
}

function SideBadge({ side }: { side: string }) {
  return <span className={`side-badge side-${side}`}>{side === "buy" ? "BUY" : "SELL"}</span>;
}

function StatusBadge({ status, working }: { status: string; working: boolean }) {
  return (
    <span className={`order-status ${working ? "order-status-working" : statusTone(status)}`}>
      {status.replaceAll("_", " ")}
    </span>
  );
}

function statusTone(status: string) {
  const value = status.toLowerCase();
  if (value.includes("fill")) return "order-status-filled";
  if (value.includes("cancel") || value.includes("expire")) return "order-status-canceled";
  if (value.includes("reject") || value.includes("fail")) return "order-status-rejected";
  return "";
}

function EmptyRow({ text }: { text: string }) {
  return <div className="terminal-empty">{text}</div>;
}

function capitalize(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
