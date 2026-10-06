import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { RefreshCw, Search, WifiOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { fetchMarketSnapshots } from "@/lib/market-api";

const priceFormatter = new Intl.NumberFormat("en-US", {
  style: "currency",
  currency: "USD",
  minimumFractionDigits: 2,
});

const volumeFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

export function MarketOverview() {
  const [search, setSearch] = useState("");
  const { data, error, isPending, isFetching, refetch } = useQuery({
    queryKey: ["market", "sp500-top-30"],
    queryFn: fetchMarketSnapshots,
    staleTime: 10_000,
    refetchInterval: 15_000,
    retry: 1,
  });

  const snapshots = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return data?.data ?? [];
    return (data?.data ?? []).filter((snapshot) =>
      `${snapshot.symbol} ${snapshot.name}`.toLowerCase().includes(query),
    );
  }, [data, search]);

  return (
    <section className="panel market-panel" aria-busy={isPending}>
      <div className="panel-heading flex-wrap gap-4">
        <div>
          <div className="label flex items-center gap-2 text-primary">
            <span className="live-pulse size-1.5 rounded-full bg-primary" /> ALPACA MARKET DATA
          </div>
          <h2 className="mt-2 text-lg font-semibold">S&amp;P 500 large-cap watchlist</h2>
          <p className="mt-2 max-w-2xl text-xs leading-5 text-muted-foreground">
            Latest eligible trade and daily session data for 30 leading S&amp;P 500 companies.
            Quotes refresh every 15 seconds.
          </p>
        </div>
        <div className="flex w-full items-center gap-2 sm:w-auto">
          <div className="relative min-w-0 flex-1 sm:w-56 sm:flex-none">
            <Search size={15} className="absolute left-3 top-2.5 text-muted-foreground" />
            <Input
              aria-label="Search market symbols"
              placeholder="Search symbol or company"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              className="h-9 border-border bg-secondary/40 pl-9 text-xs"
            />
          </div>
          <Button
            variant="outline"
            size="icon"
            className="size-9 shrink-0"
            aria-label="Refresh market data"
            title="Refresh market data"
            disabled={isFetching}
            onClick={() => void refetch()}
          >
            <RefreshCw size={15} className={isFetching ? "animate-spin" : ""} />
          </Button>
        </div>
      </div>

      {error ? (
        <div className="mt-6 flex min-h-52 flex-col items-center justify-center rounded-md border border-destructive/30 bg-destructive/5 px-6 text-center">
          <WifiOff size={24} className="text-destructive" />
          <p role="alert" className="mt-3 text-sm font-medium">
            {error instanceof Error ? error.message : "Live market data is unavailable."}
          </p>
          <Button variant="outline" size="sm" className="mt-4" onClick={() => void refetch()}>
            Try again
          </Button>
        </div>
      ) : (
        <div className="mt-6 overflow-x-auto">
          <div className="market-table min-w-[820px]" role="table" aria-label="Live market data">
            <div className="market-row market-head" role="row">
              <span role="columnheader">Company</span>
              <span role="columnheader">Last</span>
              <span role="columnheader">Change</span>
              <span role="columnheader">Open</span>
              <span role="columnheader">Day range</span>
              <span role="columnheader">Volume</span>
            </div>
            {isPending
              ? Array.from({ length: 8 }, (_, index) => (
                  <div className="market-row market-entry" role="row" key={`loading-${index}`}>
                    {Array.from({ length: 6 }, (_, cell) => (
                      <span className="h-3 animate-pulse rounded bg-secondary" key={cell} />
                    ))}
                  </div>
                ))
              : snapshots.map((snapshot) => {
                  const positive = snapshot.change >= 0;
                  return (
                    <div className="market-row market-entry" role="row" key={snapshot.symbol}>
                      <span role="cell" className="min-w-0">
                        <span className="block text-xs font-semibold">{snapshot.symbol}</span>
                        <span className="mt-1 block truncate text-[10px] text-muted-foreground">
                          {snapshot.name}
                        </span>
                      </span>
                      <span role="cell" className="text-xs font-semibold tabular-nums">
                        {snapshot.available ? priceFormatter.format(snapshot.price) : "Unavailable"}
                      </span>
                      <span
                        role="cell"
                        className={`text-xs tabular-nums ${positive ? "text-primary" : "text-destructive"}`}
                      >
                        {snapshot.available
                          ? `${positive ? "+" : ""}${snapshot.change.toFixed(2)} (${positive ? "+" : ""}${snapshot.change_percent.toFixed(2)}%)`
                          : "—"}
                      </span>
                      <span role="cell" className="text-xs tabular-nums text-muted-foreground">
                        {snapshot.open > 0 ? priceFormatter.format(snapshot.open) : "—"}
                      </span>
                      <span role="cell" className="text-xs tabular-nums text-muted-foreground">
                        {snapshot.low > 0 && snapshot.high > 0
                          ? `${priceFormatter.format(snapshot.low)} – ${priceFormatter.format(snapshot.high)}`
                          : "—"}
                      </span>
                      <span role="cell" className="text-xs tabular-nums text-muted-foreground">
                        {snapshot.volume > 0 ? volumeFormatter.format(snapshot.volume) : "—"}
                      </span>
                    </div>
                  );
                })}
            {!isPending && snapshots.length === 0 && (
              <div className="py-12 text-center text-sm text-muted-foreground">
                No companies match your search.
              </div>
            )}
          </div>
        </div>
      )}

      <div className="mt-5 flex flex-wrap items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span>
          {data
            ? `${data.count} symbols · ${data.feed.toUpperCase()} feed`
            : "Connecting to Alpaca"}
        </span>
        <span>
          {data
            ? `Provider snapshot ${new Date(data.as_of).toLocaleString()}`
            : "Waiting for the first snapshot"}
        </span>
      </div>
    </section>
  );
}
