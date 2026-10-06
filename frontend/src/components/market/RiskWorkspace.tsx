import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import {
  Activity,
  ArrowDownRight,
  ArrowLeft,
  ArrowRight,
  ArrowUpRight,
  Bell,
  BrainCircuit,
  ChartNoAxesCombined,
  ChevronDown,
  CircleHelp,
  Command,
  Flame,
  LayoutDashboard,
  LogOut,
  Menu,
  Search,
  ShieldAlert,
  SlidersHorizontal,
  Sparkles,
  TestTubeDiagonal,
  TrendingUp,
  X,
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
import { Input } from "@/components/ui/input";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { SignalDetail } from "./SignalDetail";
import { MarketOverview } from "./MarketOverview";
import { NewsFeed } from "./NewsFeed";
import {
  initialEvents,
  marketIndices,
  sentimentHistory,
  severityRank,
  type RiskEvent,
} from "@/lib/market";
import { supabase } from "@/integrations/supabase/client";
import { useQueryClient } from "@tanstack/react-query";

const navItems = [
  { label: "Overview", icon: LayoutDashboard },
  { label: "Live risk feed", icon: Activity },
  { label: "Market overview", icon: ChartNoAxesCombined },
  { label: "Portfolio risk", icon: ShieldAlert },
  { label: "Stress testing", icon: TestTubeDiagonal },
  { label: "Historical analytics", icon: TrendingUp },
];

export function RiskWorkspace({ demo = false }: { demo?: boolean }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [view, setView] = useState("Overview");
  const [events, setEvents] = useState(initialEvents);
  const [selected, setSelected] = useState<RiskEvent | null>(null);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState("All events");
  const [notifications, setNotifications] = useState(false);
  const [stress, setStress] = useState(20);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [profileName, setProfileName] = useState("Analyst");
  const [activeTicker, setActiveTicker] = useState<"NVDA" | "AAPL" | "MSFT">("NVDA");
  useEffect(() => {
    if (demo) return;
    let alive = true;
    supabase.auth.getUser().then(async ({ data }) => {
      if (!alive || !data.user) return;
      const name =
        data.user.user_metadata?.["full_name"] || data.user.email?.split("@")[0] || "Analyst";
      setProfileName(name);
      const { data: existing } = await supabase
        .from("profiles")
        .select("id")
        .eq("id", data.user.id)
        .maybeSingle();
      if (alive && !existing)
        await supabase.from("profiles").upsert({ id: data.user.id, display_name: name });
    });
    return () => {
      alive = false;
    };
  }, [demo]);
  useEffect(() => {
    const timer = window.setInterval(() => {
      setEvents((previous) => {
        const index = Math.floor(Math.random() * previous.length);
        return previous.map((event, i) =>
          i === index
            ? {
                ...event,
                time: "JUST NOW",
                sentiment: Math.max(
                  -1,
                  Math.min(
                    1,
                    Math.round((event.sentiment + (Math.random() - 0.5) * 0.08) * 100) / 100,
                  ),
                ),
              }
            : event,
        );
      });
    }, 6500);
    return () => window.clearInterval(timer);
  }, []);
  const shownEvents = useMemo(
    () =>
      events
        .filter(
          (event) =>
            `${event.ticker} ${event.company} ${event.headline} ${event.eventType}`
              .toLowerCase()
              .includes(search.toLowerCase()) &&
            (filter === "All events" || event.severity === filter),
        )
        .sort((a, b) =>
          filter === "All events"
            ? a.id - b.id
            : severityRank[b.severity] - severityRank[a.severity],
        ),
    [events, search, filter],
  );
  const onSignOut = async () => {
    await queryClient.cancelQueries();
    queryClient.clear();
    await supabase.auth.signOut();
    navigate({ to: "/auth", search: { mode: "login" }, replace: true });
  };
  const Sidebar = () => (
    <div className="flex h-full flex-col">
      <Link to="/" className="brand flex items-center gap-2.5 px-7 py-8">
        <span className="brand-mark">
          <Activity size={17} strokeWidth={2.5} />
        </span>
        <span>
          Trading <span className="text-primary">wens</span>
        </span>
      </Link>
      <div className="px-7 pt-5 pb-3 label text-muted-foreground">WORKSPACE</div>
      <nav className="space-y-1 px-3">
        {navItems.map((item) => (
          <Button
            key={item.label}
            variant="ghost"
            className={`sidebar-item ${view === item.label ? "sidebar-active" : ""}`}
            onClick={() => {
              setView(item.label);
              setMobileOpen(false);
            }}
          >
            <item.icon size={17} />
            <span>{item.label}</span>
            {view === item.label && <span className="ml-auto size-1.5 rounded-full bg-primary" />}
          </Button>
        ))}
      </nav>
      <div className="mt-8 px-7 pb-3 label text-muted-foreground">MONITORING</div>
      <div className="space-y-1 px-3">
        <Button
          variant="ghost"
          className={`sidebar-item ${view === "AI Risk Signals" ? "sidebar-active" : ""}`}
          onClick={() => {
            setView("AI Risk Signals");
            setMobileOpen(false);
          }}
        >
          <BrainCircuit size={17} />
          AI Risk Signals
        </Button>
        <Button
          variant="ghost"
          className={`sidebar-item ${view === "Alerts" ? "sidebar-active" : ""}`}
          onClick={() => {
            setView("Alerts");
            setMobileOpen(false);
          }}
        >
          <Bell size={17} />
          Alerts{" "}
          <span className="ml-auto rounded bg-destructive/15 px-1.5 py-0.5 text-[10px] text-destructive">
            3
          </span>
        </Button>
      </div>
      <div className="mt-auto px-5 pb-5 pt-8">
        <div className="rounded-md border border-primary/20 bg-primary/5 p-4">
          <div className="flex items-center gap-2 text-xs font-semibold text-primary">
            <Sparkles size={14} /> {demo ? "DEMO ENVIRONMENT" : "SIMULATED WORKSPACE"}
          </div>
          <p className="mt-2 text-xs leading-5 text-muted-foreground">
            {demo
              ? "Explore with simulated market events. No account required."
              : "Explore market scenarios with illustrative signals."}
          </p>
          {demo && (
            <Button asChild size="sm" className="mt-4 w-full">
              <Link to="/auth" search={{ mode: "signup" }}>
                Create account <ArrowRight size={13} />
              </Link>
            </Button>
          )}
        </div>
      </div>
    </div>
  );
  const viewingFeed =
    view === "Overview" ||
    view === "Live risk feed" ||
    view === "AI Risk Signals" ||
    view === "Alerts";
  const showingLiveMarket = !demo && view === "Market overview";
  const showingLiveNews = !demo && view === "Live risk feed";
  const showingProviderData = showingLiveMarket || showingLiveNews;
  return (
    <div className="workspace min-h-screen bg-background text-foreground">
      <aside className="workspace-sidebar hidden lg:block">
        <Sidebar />
      </aside>
      <div className="workspace-main">
        <header className="workspace-header">
          <div className="flex min-w-0 items-center gap-4">
            <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
              <SheetTrigger asChild>
                <Button size="icon" variant="ghost" className="lg:hidden" aria-label="Open menu">
                  <Menu />
                </Button>
              </SheetTrigger>
              <SheetContent side="left" className="w-72 border-border bg-background p-0">
                <SheetTitle className="sr-only">Navigation</SheetTitle>
                <Sidebar />
              </SheetContent>
            </Sheet>
            <div className="hidden items-center gap-2 text-xs text-muted-foreground sm:flex">
              <span>Workspace</span>
              <span>/</span>
              <span className="text-foreground">{view}</span>
            </div>
            <span className="sm:hidden text-sm font-semibold">{view}</span>
          </div>
          <div className="flex items-center gap-3 sm:gap-5">
            <span className="hidden items-center gap-2 text-[11px] font-medium text-primary sm:flex">
              <span className="live-pulse size-1.5 rounded-full bg-primary" /> SYSTEM LIVE
            </span>
            <span className="hidden h-5 w-px bg-border sm:block" />
            <div className="relative">
              <Button
                variant="ghost"
                size="icon"
                aria-label="Notifications"
                onClick={() => setNotifications(!notifications)}
              >
                <Bell size={18} />
                <span className="absolute right-2 top-1.5 size-1.5 rounded-full bg-destructive" />
              </Button>
              {notifications && (
                <div className="notification-popover">
                  <div className="label mb-4 text-muted-foreground">RECENT ALERTS</div>
                  {events
                    .filter((e) => e.severity === "Critical" || e.severity === "High")
                    .slice(0, 3)
                    .map((e) => (
                      <Button
                        variant="ghost"
                        key={e.id}
                        className="h-auto w-full justify-start whitespace-normal border-t border-border px-0 py-3 text-left text-xs leading-5"
                        onClick={() => {
                          setSelected(e);
                          setNotifications(false);
                        }}
                      >
                        {e.ticker} · {e.headline}
                      </Button>
                    ))}
                </div>
              )}
            </div>
            <span className="h-5 w-px bg-border" />
            {demo ? (
              <Button asChild size="sm" variant="outline">
                <Link to="/auth">
                  Sign in <ArrowRight size={14} />
                </Link>
              </Button>
            ) : (
              <div className="flex items-center gap-2">
                <div className="flex size-8 items-center justify-center rounded-full bg-primary/15 text-xs font-semibold text-primary">
                  {profileName[0]?.toUpperCase()}
                </div>
                <span className="hidden max-w-24 truncate text-xs sm:block">{profileName}</span>
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label="Sign out"
                  title="Sign out"
                  onClick={onSignOut}
                >
                  <LogOut size={16} />
                </Button>
              </div>
            )}
          </div>
        </header>
        <main className="workspace-content">
          <div className="demo-banner">
            <span className="flex items-center gap-2">
              <span className="size-1.5 rounded-full bg-accent" />{" "}
              {showingLiveMarket
                ? "AUTHENTICATED WORKSPACE · ALPACA MARKET DATA"
                : showingLiveNews
                  ? "AUTHENTICATED WORKSPACE · GDELT MARKET NEWS"
                  : `${demo ? "PUBLIC DEMO" : "WORKSPACE"} · SIMULATED DATA`}
            </span>
            {demo && (
              <Link
                to="/auth"
                search={{ mode: "signup" }}
                className="text-foreground underline underline-offset-4"
              >
                Unlock your workspace <ArrowRight size={12} className="inline" />
              </Link>
            )}
          </div>
          <div className="page-intro">
            <div>
              <div className="label flex items-center gap-2 text-primary">
                <span className="size-1 rounded-full bg-primary" />{" "}
                {showingLiveMarket
                  ? "LIVE MARKET OVERVIEW"
                  : showingLiveNews
                    ? "LIVE NEWS MONITOR"
                    : "REAL-TIME INTELLIGENCE"}
              </div>
              <h1 className="mt-3 text-3xl font-semibold sm:text-4xl">
                {view === "Overview" ? "Market intelligence" : view}
              </h1>
              <p className="mt-2 text-sm text-muted-foreground">
                {view === "Overview"
                  ? "A clearer view of what moves your portfolio."
                  : showingLiveMarket
                    ? "Latest quotes for a maintained universe of leading S&P 500 companies."
                    : showingLiveNews
                      ? "Recent global coverage connected to the companies in your watchlist."
                      : view === "Stress testing"
                        ? "Explore how market shocks could affect your positions."
                        : view === "Portfolio risk"
                          ? "Track exposure, concentration, and potential downside."
                          : "Signals from across the market, distilled into perspective."}
              </p>
            </div>
            <div className="text-left sm:text-right">
              <div className="label text-muted-foreground">MARKET STATUS</div>
              <div className="mt-2 flex items-center gap-2 text-xs text-primary sm:justify-end">
                <span className="size-1.5 rounded-full bg-primary" /> Monitoring active
              </div>
            </div>
          </div>
          {showingLiveMarket && <MarketOverview />}
          {showingLiveNews && <NewsFeed />}
          {!showingProviderData && (
            <div className="index-strip">
              {marketIndices.map((index) => (
                <div className="index-item" key={index.name}>
                  <div className="label text-muted-foreground">{index.name}</div>
                  <div className="mt-2 flex items-baseline gap-2">
                    <span className="text-lg font-semibold tabular-nums">{index.value}</span>
                    <span
                      className={`text-xs tabular-nums ${index.positive ? "text-primary" : "text-destructive"}`}
                    >
                      {index.change}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
          {!showingProviderData && (
            <div className="metric-grid">
              <div className="metric">
                <div className="flex justify-between">
                  <span className="label text-muted-foreground">ACTIVE SIGNALS</span>
                  <Activity size={17} className="text-primary" />
                </div>
                <div className="mt-5 flex items-end gap-3">
                  <span className="text-4xl font-semibold tabular-nums">
                    {events.length * 21 + 2}
                  </span>
                  <span className="mb-1 flex items-center text-xs text-primary">
                    <ArrowUpRight size={14} /> 12.8%
                  </span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">Across monitored markets</p>
              </div>
              <div className="metric">
                <div className="flex justify-between">
                  <span className="label text-muted-foreground">HIGH-RISK EVENTS</span>
                  <Flame size={17} className="text-destructive" />
                </div>
                <div className="mt-5 flex items-end gap-3">
                  <span className="text-4xl font-semibold tabular-nums">
                    {events.filter((e) => severityRank[e.severity] >= 3).length}
                  </span>
                  <span className="mb-1 text-xs text-destructive">Needs attention</span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">In your tracked universe</p>
              </div>
              <div className="metric">
                <div className="flex justify-between">
                  <span className="label text-muted-foreground">MARKET SENTIMENT</span>
                  <Sparkles size={17} className="text-accent" />
                </div>
                <div className="mt-5 flex items-end gap-3">
                  <span className="text-4xl font-semibold tabular-nums">+0.28</span>
                  <span className="mb-1 flex items-center text-xs text-primary">
                    <ArrowUpRight size={14} /> 0.06
                  </span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">Weighted signal average</p>
              </div>
              <div className="metric">
                <div className="flex justify-between">
                  <span className="label text-muted-foreground">PORTFOLIO EXPOSURE</span>
                  <ShieldAlert size={17} className="text-accent" />
                </div>
                <div className="mt-5 flex items-end gap-3">
                  <span className="text-4xl font-semibold tabular-nums">$874k</span>
                  <span className="mb-1 flex items-center text-xs text-destructive">
                    <ArrowDownRight size={14} /> 2.4%
                  </span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">Illustrative tracked positions</p>
              </div>
            </div>
          )}
          {!showingProviderData && (
            <div className="analysis-grid">
              <section className="panel chart-panel">
                <div className="panel-heading">
                  <div>
                    <div className="label text-muted-foreground">MARKET PULSE</div>
                    <h2 className="mt-2 text-lg font-semibold">Sentiment trajectory</h2>
                  </div>
                  <div className="flex gap-1 rounded-md border border-border p-1">
                    {(["NVDA", "AAPL", "MSFT"] as const).map((ticker) => (
                      <Button
                        key={ticker}
                        variant="ghost"
                        size="sm"
                        className={`h-7 px-2 text-[11px] ${activeTicker === ticker ? "bg-secondary text-foreground" : "text-muted-foreground"}`}
                        onClick={() => setActiveTicker(ticker)}
                      >
                        {ticker}
                      </Button>
                    ))}
                  </div>
                </div>
                <div className="mt-6 h-56 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart
                      data={sentimentHistory}
                      margin={{ top: 5, right: 0, left: -32, bottom: 0 }}
                    >
                      <defs>
                        <linearGradient id="sentimentFill" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor="var(--primary)" stopOpacity={0.25} />
                          <stop offset="100%" stopColor="var(--primary)" stopOpacity={0} />
                        </linearGradient>
                      </defs>
                      <CartesianGrid
                        vertical={false}
                        stroke="var(--border)"
                        strokeDasharray="3 5"
                      />
                      <XAxis
                        dataKey="hour"
                        tickLine={false}
                        axisLine={false}
                        tick={{ fill: "var(--muted-foreground)", fontSize: 10 }}
                        interval={1}
                      />
                      <YAxis
                        tickLine={false}
                        axisLine={false}
                        tick={{ fill: "var(--muted-foreground)", fontSize: 10 }}
                        domain={[0, 100]}
                      />
                      <Tooltip
                        contentStyle={{
                          background: "var(--popover)",
                          border: "1px solid var(--border)",
                          borderRadius: 4,
                          color: "var(--foreground)",
                          fontSize: 12,
                        }}
                      />
                      <Area
                        type="monotone"
                        dataKey={activeTicker}
                        stroke="var(--primary)"
                        fill="url(#sentimentFill)"
                        strokeWidth={2}
                        animationDuration={500}
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
                <div className="mt-4 flex items-center justify-between text-xs text-muted-foreground">
                  <span>Sentiment index · 0–100</span>
                  <span className="flex items-center gap-2">
                    <span className="size-2 rounded-full bg-primary" /> {activeTicker}
                  </span>
                </div>
              </section>
              <section className="panel heatmap-panel">
                <div className="label text-muted-foreground">CROSS-ASSET VIEW</div>
                <h2 className="mt-2 text-lg font-semibold">Sentiment heatmap</h2>
                <div className="mt-6 grid grid-cols-3 gap-2">
                  {events.map((e) => (
                    <Button
                      key={e.id}
                      variant="ghost"
                      className={`heat-cell h-20 flex-col gap-1 ${e.sentiment >= 0.4 ? "heat-positive" : e.sentiment <= -0.5 ? "heat-negative" : "heat-neutral"}`}
                      onClick={() => setSelected(e)}
                    >
                      <span className="text-sm font-semibold">{e.ticker}</span>
                      <span className="text-xs tabular-nums">
                        {e.sentiment > 0 ? "+" : ""}
                        {e.sentiment.toFixed(2)}
                      </span>
                    </Button>
                  ))}
                </div>
                <div className="mt-5 flex items-center justify-between text-[10px] text-muted-foreground">
                  <span>NEGATIVE</span>
                  <div className="flex gap-1">
                    <span className="h-2 w-8 bg-destructive/60" />
                    <span className="h-2 w-8 bg-accent/50" />
                    <span className="h-2 w-8 bg-primary/60" />
                  </div>
                  <span>POSITIVE</span>
                </div>
              </section>
            </div>
          )}
          {view === "Stress testing" && (
            <section className="panel mt-5">
              <div className="panel-heading">
                <div>
                  <div className="label text-muted-foreground">SCENARIO SIMULATOR</div>
                  <h2 className="mt-2 text-lg font-semibold">Market downturn</h2>
                </div>
                <TestTubeDiagonal className="text-primary" size={20} />
              </div>
              <div className="mt-6 grid gap-6 sm:grid-cols-[1fr_auto] sm:items-center">
                <div>
                  <label htmlFor="stress-range" className="text-sm">
                    Index decline: <strong>{stress}%</strong>
                  </label>
                  <input
                    id="stress-range"
                    type="range"
                    min="5"
                    max="50"
                    step="5"
                    value={stress}
                    onChange={(e) => setStress(Number(e.target.value))}
                    className="mt-4 w-full accent-primary"
                  />
                </div>
                <div className="min-w-44 rounded border border-destructive/30 bg-destructive/10 p-4">
                  <div className="label text-muted-foreground">EST. PORTFOLIO IMPACT</div>
                  <div className="mt-2 text-xl font-semibold text-destructive">
                    -${Math.round(((874400 * stress) / 100) * 0.78).toLocaleString()}
                  </div>
                </div>
              </div>
              <p className="mt-5 text-xs text-muted-foreground">
                Illustrative scenario, not a forecast or investment advice.
              </p>
            </section>
          )}
          {view === "Portfolio risk" && (
            <section className="panel mt-5">
              <div className="label text-muted-foreground">EXPOSURE BREAKDOWN</div>
              <h2 className="mt-2 text-lg font-semibold">Monitored positions</h2>
              <div className="mt-6 space-y-4">
                {events.slice(0, 5).map((e) => (
                  <div key={e.id} className="flex items-center gap-4">
                    <span className="w-12 text-xs font-semibold">{e.ticker}</span>
                    <div className="h-2 flex-1 overflow-hidden rounded-full bg-secondary">
                      <div
                        className="h-full rounded-full bg-primary"
                        style={{
                          width: `${Math.min(100, parseInt(e.exposure.replace(/\D/g, "")) / 2500)}%`,
                        }}
                      />
                    </div>
                    <span className="w-20 text-right text-xs tabular-nums text-muted-foreground">
                      {e.exposure}
                    </span>
                  </div>
                ))}
              </div>
            </section>
          )}
          {!showingProviderData && (
            <section className="panel feed-panel mt-5">
              <div className="panel-heading flex-wrap gap-4">
                <div>
                  <div className="label flex items-center gap-2 text-primary">
                    <span className="live-pulse size-1.5 rounded-full bg-primary" /> LIVE MONITORING
                  </div>
                  <h2 className="mt-2 text-lg font-semibold">
                    {view === "Alerts"
                      ? "Priority alerts"
                      : viewingFeed
                        ? "Live risk feed"
                        : "Latest signals"}
                  </h2>
                </div>
                <div className="flex w-full flex-wrap gap-2 sm:w-auto">
                  <div className="relative min-w-0 flex-1 sm:w-52 sm:flex-none">
                    <Search size={15} className="absolute left-3 top-2.5 text-muted-foreground" />
                    <Input
                      aria-label="Search company or event"
                      placeholder="Search company or event"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      className="h-9 border-border bg-secondary/40 pl-9 text-xs"
                    />
                  </div>
                  <div className="relative">
                    <SlidersHorizontal
                      size={14}
                      className="pointer-events-none absolute left-3 top-2.5 text-muted-foreground"
                    />
                    <select
                      aria-label="Filter severity"
                      value={filter}
                      onChange={(e) => setFilter(e.target.value)}
                      className="h-9 appearance-none rounded-md border border-border bg-secondary/40 pl-9 pr-8 text-xs text-foreground"
                    >
                      <option>All events</option>
                      <option>Critical</option>
                      <option>High</option>
                      <option>Moderate</option>
                      <option>Low</option>
                    </select>
                    <ChevronDown
                      size={13}
                      className="pointer-events-none absolute right-2 top-3 text-muted-foreground"
                    />
                  </div>
                </div>
              </div>
              <div className="mt-5 overflow-x-auto">
                <div className="feed-table min-w-[670px]">
                  <div className="feed-row feed-head">
                    <span>EVENT / SOURCE</span>
                    <span>ASSET</span>
                    <span>SENTIMENT</span>
                    <span>IMPACT</span>
                    <span>SEVERITY</span>
                    <span></span>
                  </div>
                  {shownEvents
                    .filter((e) => view !== "Alerts" || severityRank[e.severity] >= 3)
                    .map((event) => (
                      <Button
                        variant="ghost"
                        key={event.id}
                        className={`feed-row feed-entry ${selected?.id === event.id ? "feed-selected" : ""}`}
                        onClick={() => setSelected(event)}
                      >
                        <span className="min-w-0 text-left">
                          <span className="block truncate text-xs font-medium">
                            {event.headline}
                          </span>
                          <span className="mt-2 block text-[10px] text-muted-foreground">
                            {event.source} <span className="mx-1">·</span> {event.time}
                          </span>
                        </span>
                        <span className="text-left">
                          <span className="block text-xs font-semibold">{event.ticker}</span>
                          <span className="mt-1 block truncate text-[10px] text-muted-foreground">
                            {event.company}
                          </span>
                        </span>
                        <span
                          className={`text-left text-xs tabular-nums ${event.sentiment < 0 ? "text-destructive" : "text-primary"}`}
                        >
                          {event.sentiment > 0 ? "+" : ""}
                          {event.sentiment.toFixed(2)}
                        </span>
                        <span className="text-left text-xs tabular-nums">
                          {event.impact}
                          <span className="text-muted-foreground">/10</span>
                        </span>
                        <span className={`severity severity-${event.severity.toLowerCase()}`}>
                          {event.severity}
                        </span>
                        <ArrowRight size={15} className="text-muted-foreground" />
                      </Button>
                    ))}
                  {shownEvents.length === 0 && (
                    <div className="py-12 text-center text-sm text-muted-foreground">
                      No signals match your search.
                    </div>
                  )}
                </div>
              </div>
              <div className="mt-5 flex items-center justify-between text-xs text-muted-foreground">
                <span>{shownEvents.length} signals shown</span>
                <span>Updates every few seconds</span>
              </div>
            </section>
          )}
          <footer className="mt-7 flex flex-wrap items-center justify-between gap-3 pb-7 text-[11px] text-muted-foreground">
            <span>
              © TRADING WENS ·{" "}
              {showingLiveMarket
                ? "DATA PROVIDED BY ALPACA"
                : showingLiveNews
                  ? "NEWS PROVIDED BY GDELT"
                  : "MARKET SIMULATION"}
            </span>
            <span>For demonstration only. Not financial advice.</span>
          </footer>
        </main>
      </div>
      {selected && (
        <>
          <div className="detail-backdrop" onClick={() => setSelected(null)} />
          <aside className="detail-drawer">
            <SignalDetail event={selected} onClose={() => setSelected(null)} />
          </aside>
        </>
      )}
    </div>
  );
}
