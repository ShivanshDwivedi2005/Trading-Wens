import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { CheckCircle2, Clock3, ExternalLink, FileClock, Layers3, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { AuditTrail, FillsView, OrderBook, PositionBook } from "@/components/market/OrderMonitor";
import {
  fetchOrderMonitor,
  type OrderAuditEvent,
  type PaperFill,
  type PaperOrder,
  type TradingPortfolio,
} from "@/lib/trading-api";

type LedgerView = "positions" | "orders" | "fills" | "audit";

export function AccountExecutionOverview({ portfolio }: { portfolio: TradingPortfolio }) {
  const [view, setView] = useState<LedgerView>("positions");
  const [filter, setFilter] = useState("");
  const monitor = useQuery({
    queryKey: ["trading", "orders"],
    queryFn: fetchOrderMonitor,
    staleTime: 5_000,
    refetchInterval: 10_000,
    retry: 1,
  });
  const symbol = filter.trim().toUpperCase();
  const positions = useMemo(
    () => portfolio.positions.filter((position) => !symbol || position.symbol.includes(symbol)),
    [portfolio.positions, symbol],
  );
  const orders = useMemo(
    () =>
      (monitor.data?.orders ?? []).filter(
        (order) => order.working && (!symbol || order.symbol.includes(symbol)),
      ),
    [monitor.data?.orders, symbol],
  );
  const fills = filterBySymbol(monitor.data?.fills ?? [], symbol);
  const audit = filterBySymbol(monitor.data?.audit_trail ?? [], symbol);

  return (
    <section className="terminal-panel account-execution-overview">
      <div className="execution-titlebar">
        <div>
          <div className="label text-primary">ACCOUNT EXECUTION WORKSPACE</div>
          <h2 className="mt-1 text-base font-semibold">Positions and order activity</h2>
          <p className="mt-1 text-[11px] text-muted-foreground">
            Account-scoped provider data persisted to the execution ledger.
          </p>
        </div>
        <Button asChild variant="outline" size="sm" className="min-h-11">
          <Link to="/orders" search={{ symbol: filter.trim().toUpperCase() }}>
            Full monitor <ExternalLink size={13} aria-hidden="true" />
          </Link>
        </Button>
      </div>
      <div className="terminal-toolbar">
        <div className="terminal-tabs" role="tablist" aria-label="Account execution views">
          <LedgerTab
            active={view === "positions"}
            icon={Layers3}
            label="Positions"
            count={positions.length}
            onClick={() => setView("positions")}
          />
          <LedgerTab
            active={view === "orders"}
            icon={Clock3}
            label="Working Order Book"
            count={orders.length}
            onClick={() => setView("orders")}
          />
          <LedgerTab
            active={view === "fills"}
            icon={CheckCircle2}
            label="Fill Book"
            count={fills.length}
            onClick={() => setView("fills")}
          />
          <LedgerTab
            active={view === "audit"}
            icon={FileClock}
            label="Audit Trail"
            count={audit.length}
            onClick={() => setView("audit")}
          />
        </div>
        <div className="relative w-full sm:w-52">
          <Search
            size={14}
            aria-hidden="true"
            className="absolute left-3 top-3.5 text-muted-foreground"
          />
          <Input
            aria-label="Filter account activity by symbol"
            placeholder="Filter contract"
            value={filter}
            onChange={(event) => setFilter(event.target.value.toUpperCase())}
            className="h-11 pl-9 font-mono text-xs uppercase"
          />
        </div>
      </div>
      {monitor.error ? (
        <div className="terminal-inline-error" role="alert">
          The execution ledger could not be refreshed. Positions remain available from the current
          account snapshot.
        </div>
      ) : view !== "positions" && monitor.isPending ? (
        <div className="terminal-loading" aria-label="Loading execution ledger">
          <div className="h-10 animate-pulse rounded bg-secondary" />
          <div className="h-10 animate-pulse rounded bg-secondary" />
          <div className="h-10 animate-pulse rounded bg-secondary" />
        </div>
      ) : view === "positions" ? (
        <PositionBook positions={positions} accountID={portfolio.account.id} />
      ) : view === "orders" ? (
        <OrderBook orders={orders} />
      ) : view === "fills" ? (
        <FillsView fills={fills} positions={portfolio.positions} />
      ) : (
        <AuditTrail events={audit} />
      )}
      <div className="execution-statusbar">
        <span>ALPACA · PAPER · DATABASE PERSISTED</span>
        <span>
          {monitor.data
            ? `Ledger synchronized ${new Date(monitor.data.as_of).toLocaleTimeString()}`
            : "Synchronizing ledger"}
        </span>
      </div>
    </section>
  );
}

function LedgerTab({
  active,
  icon: Icon,
  label,
  count,
  onClick,
}: {
  active: boolean;
  icon: typeof Layers3;
  label: string;
  count: number;
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
      <Icon size={14} aria-hidden="true" /> {label} <span>{count}</span>
    </button>
  );
}

function filterBySymbol<T extends PaperFill | OrderAuditEvent | PaperOrder>(
  rows: T[],
  symbol: string,
) {
  return rows.filter((row) => !symbol || row.symbol.includes(symbol));
}
