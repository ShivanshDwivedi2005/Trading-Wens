import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { formatDistanceToNowStrict } from "date-fns";
import { ExternalLink, Newspaper, RefreshCw, Search, WifiOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { fetchMarketNews } from "@/lib/news-api";

export function NewsFeed() {
  const [search, setSearch] = useState("");
  const { data, error, isPending, isFetching, refetch } = useQuery({
    queryKey: ["news", "large-cap-market"],
    queryFn: fetchMarketNews,
    staleTime: 120_000,
    refetchInterval: 120_000,
    retry: 1,
  });
  const articles = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return data?.data ?? [];
    return (data?.data ?? []).filter((article) =>
      `${article.title} ${article.domain} ${article.matched_symbols.join(" ")}`
        .toLowerCase()
        .includes(query),
    );
  }, [data, search]);

  return (
    <section className="panel" aria-busy={isPending}>
      <div className="panel-heading flex-wrap gap-4">
        <div>
          <div className="label flex items-center gap-2 text-primary">
            <Newspaper size={14} /> GDELT GLOBAL COVERAGE
          </div>
          <h2 className="mt-2 text-lg font-semibold">Latest company news</h2>
          <p className="mt-2 max-w-2xl text-xs leading-5 text-muted-foreground">
            Recent English-language coverage matched to the large-cap watchlist. Headlines are
            provider data and are not risk scores or investment advice.
          </p>
        </div>
        <div className="flex w-full items-center gap-2 sm:w-auto">
          <div className="relative min-w-0 flex-1 sm:w-64 sm:flex-none">
            <Search size={15} className="absolute left-3 top-2.5 text-muted-foreground" />
            <Input
              aria-label="Search market news"
              placeholder="Search headlines or symbols"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              className="h-9 border-border bg-secondary/40 pl-9 text-xs"
            />
          </div>
          <Button
            variant="outline"
            size="icon"
            className="size-9 shrink-0"
            aria-label="Refresh market news"
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
            {error instanceof Error ? error.message : "Market news is unavailable."}
          </p>
          <Button variant="outline" size="sm" className="mt-4" onClick={() => void refetch()}>
            Try again
          </Button>
        </div>
      ) : (
        <div className="news-grid mt-6">
          {isPending
            ? Array.from({ length: 8 }, (_, index) => (
                <div className="news-card" key={`loading-${index}`}>
                  <div className="h-3 w-24 animate-pulse rounded bg-secondary" />
                  <div className="mt-4 h-4 animate-pulse rounded bg-secondary" />
                  <div className="mt-2 h-4 w-3/4 animate-pulse rounded bg-secondary" />
                </div>
              ))
            : articles.map((article) => (
                <article className="news-card" key={article.id}>
                  <div className="flex flex-wrap items-center gap-2 text-[10px] text-muted-foreground">
                    <span className="uppercase tracking-wider">{article.domain}</span>
                    <span aria-hidden="true">·</span>
                    <time dateTime={article.published_at}>
                      {formatDistanceToNowStrict(new Date(article.published_at), {
                        addSuffix: true,
                      })}
                    </time>
                  </div>
                  <h3 className="mt-3 text-sm font-medium leading-6">{article.title}</h3>
                  <div className="mt-5 flex items-end justify-between gap-3">
                    <div className="flex flex-wrap gap-1.5">
                      {article.matched_symbols.map((symbol) => (
                        <span className="news-symbol" key={symbol}>
                          {symbol}
                        </span>
                      ))}
                    </div>
                    <a
                      href={article.url}
                      target="_blank"
                      rel="noreferrer noopener"
                      className="inline-flex min-h-11 shrink-0 items-center gap-1.5 text-xs text-primary hover:underline"
                      aria-label={`Read ${article.title}`}
                    >
                      Read <ExternalLink size={13} />
                    </a>
                  </div>
                </article>
              ))}
          {!isPending && articles.length === 0 && (
            <div className="col-span-full py-12 text-center text-sm text-muted-foreground">
              No articles match your search.
            </div>
          )}
        </div>
      )}

      <div className="mt-5 flex flex-wrap items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span>{data ? `${data.count} recent articles` : "Connecting to GDELT"}</span>
        <span>
          {data ? `Feed checked ${new Date(data.as_of).toLocaleString()}` : "Waiting for news"}
        </span>
      </div>
    </section>
  );
}
