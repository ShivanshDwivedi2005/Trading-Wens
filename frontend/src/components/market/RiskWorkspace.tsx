import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import {
  Activity,
  ArrowRight,
  Bell,
  BrainCircuit,
  BriefcaseBusiness,
  ChartCandlestick,
  LayoutDashboard,
  LogOut,
  Menu,
  Search,
  ShieldAlert,
  SlidersHorizontal,
  Sparkles,
} from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { fetchCurrentUser, signOut } from "@/lib/auth-api";
import { initialEvents, severityRank, type RiskEvent } from "@/lib/market";
import { LiveTrading } from "./LiveTrading";
import { AccountOverview, LiveRiskFeed, PortfolioStatus } from "./PortfolioViews";
import { SignalDetail } from "./SignalDetail";

const navItems = [
  { label: "Overview", icon: LayoutDashboard },
  { label: "Live risk feed", icon: Activity },
  { label: "Portfolio status", icon: BriefcaseBusiness },
  { label: "Live Trading", icon: ChartCandlestick },
];

const primaryViews = new Set(navItems.map((item) => item.label));

export function RiskWorkspace({ demo = false }: { demo?: boolean }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [view, setView] = useState(demo ? "AI Risk Signals" : "Overview");
  const [events, setEvents] = useState(initialEvents);
  const [selected, setSelected] = useState<RiskEvent | null>(null);
  const [notifications, setNotifications] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);
  const [profileName, setProfileName] = useState("Analyst");

  useEffect(() => {
    if (demo) return;
    let alive = true;
    fetchCurrentUser().then((user) => {
      if (!alive || !user) return;
      setProfileName(user.display_name || user.email.split("@")[0] || "Analyst");
    });
    return () => {
      alive = false;
    };
  }, [demo]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setEvents((previous) => {
        const index = Math.floor(Math.random() * previous.length);
        return previous.map((event, eventIndex) =>
          eventIndex === index
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

  const onSignOut = async () => {
    await queryClient.cancelQueries();
    queryClient.clear();
    await signOut();
    navigate({ to: "/auth", search: { mode: "login" }, replace: true });
  };

  const chooseView = (nextView: string) => {
    setView(nextView);
    setMobileOpen(false);
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
      <div className="px-7 pb-3 pt-5 label text-muted-foreground">WORKSPACE</div>
      <nav className="space-y-1 px-3" aria-label="Workspace">
        {navItems.map((item) => (
          <Button
            key={item.label}
            variant="ghost"
            className={`sidebar-item ${view === item.label ? "sidebar-active" : ""}`}
            onClick={() => chooseView(item.label)}
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
          onClick={() => chooseView("AI Risk Signals")}
        >
          <BrainCircuit size={17} /> AI Risk Signals
        </Button>
        <Button
          variant="ghost"
          className={`sidebar-item ${view === "Alerts" ? "sidebar-active" : ""}`}
          onClick={() => chooseView("Alerts")}
        >
          <Bell size={17} /> Alerts
          <span className="ml-auto rounded bg-destructive/15 px-1.5 py-0.5 text-[10px] text-destructive">
            3
          </span>
        </Button>
      </div>
      <div className="mt-auto px-5 pb-5 pt-8">
        <div className="rounded-md border border-primary/20 bg-primary/5 p-4">
          <div className="flex items-center gap-2 text-xs font-semibold text-primary">
            <Sparkles size={14} /> {demo ? "DEMO ENVIRONMENT" : "ALPACA PAPER ACCOUNT"}
          </div>
          <p className="mt-2 text-xs leading-5 text-muted-foreground">
            {demo
              ? "Explore simulated monitoring signals. Sign in for live account data."
              : "Portfolio data and orders use the configured Alpaca paper account."}
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

  const isProviderView = primaryViews.has(view) && !demo;
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
            <span className="truncate text-sm font-semibold sm:hidden">{view}</span>
          </div>
          <div className="flex items-center gap-3 sm:gap-5">
            <span className="hidden items-center gap-2 text-[11px] font-medium text-primary sm:flex">
              <span className="live-pulse size-1.5 rounded-full bg-primary" /> SYSTEM LIVE
            </span>
            <div className="relative">
              <Button
                variant="ghost"
                size="icon"
                aria-label="Notifications"
                aria-expanded={notifications}
                onClick={() => setNotifications(!notifications)}
              >
                <Bell size={18} />
                <span className="absolute right-2 top-1.5 size-1.5 rounded-full bg-destructive" />
              </Button>
              {notifications && (
                <div className="notification-popover">
                  <div className="label mb-4 text-muted-foreground">RECENT ALERTS</div>
                  {events
                    .filter((event) => severityRank[event.severity] >= 3)
                    .slice(0, 3)
                    .map((event) => (
                      <Button
                        variant="ghost"
                        key={event.id}
                        className="h-auto w-full justify-start whitespace-normal border-t border-border px-0 py-3 text-left text-xs leading-5"
                        onClick={() => {
                          setSelected(event);
                          setNotifications(false);
                        }}
                      >
                        {event.ticker} · {event.headline}
                      </Button>
                    ))}
                </div>
              )}
            </div>
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
                <Button size="icon" variant="ghost" aria-label="Sign out" onClick={onSignOut}>
                  <LogOut size={16} />
                </Button>
              </div>
            )}
          </div>
        </header>

        <main className="workspace-content">
          <div className="demo-banner">
            <span className="flex items-center gap-2">
              <span className="size-1.5 rounded-full bg-accent" />
              {isProviderView
                ? "ALPACA PAPER ACCOUNT · LIVE PROVIDER DATA"
                : "MONITORING · SIMULATED SIGNALS"}
            </span>
            {demo && (
              <Link
                to="/auth"
                search={{ mode: "signup" }}
                className="text-foreground underline underline-offset-4"
              >
                Unlock live workspace <ArrowRight size={12} className="inline" />
              </Link>
            )}
          </div>
          <div className="page-intro">
            <div>
              <div className="label flex items-center gap-2 text-primary">
                <span className="size-1 rounded-full bg-primary" />
                {isProviderView ? "LIVE ACCOUNT WORKSPACE" : "RISK MONITORING"}
              </div>
              <h1 className="mt-3 text-3xl font-semibold sm:text-4xl">{view}</h1>
              <p className="mt-2 text-sm text-muted-foreground">{viewDescription(view)}</p>
            </div>
            <div className="text-left sm:text-right">
              <div className="label text-muted-foreground">ACCOUNT MODE</div>
              <div className="mt-2 flex items-center gap-2 text-xs text-primary sm:justify-end">
                <span className="size-1.5 rounded-full bg-primary" />
                {isProviderView ? "Paper trading active" : "Monitoring active"}
              </div>
            </div>
          </div>

          {!demo && view === "Overview" && <AccountOverview />}
          {!demo && view === "Live risk feed" && <LiveRiskFeed />}
          {!demo && view === "Portfolio status" && <PortfolioStatus />}
          {!demo && view === "Live Trading" && <LiveTrading />}
          {demo && primaryViews.has(view) && <DemoLocked />}
          {(view === "AI Risk Signals" || view === "Alerts") && (
            <MonitoringSignals
              events={events}
              alertOnly={view === "Alerts"}
              onSelect={setSelected}
            />
          )}

          <footer className="mt-7 flex flex-wrap items-center justify-between gap-3 pb-7 text-[11px] text-muted-foreground">
            <span>
              © TRADING WENS · {isProviderView ? "ALPACA PAPER TRADING" : "SIMULATED MONITORING"}
            </span>
            <span>Research interface only. Not financial advice.</span>
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

function MonitoringSignals({
  events,
  alertOnly,
  onSelect,
}: {
  events: RiskEvent[];
  alertOnly: boolean;
  onSelect: (event: RiskEvent) => void;
}) {
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState("All events");
  const shown = useMemo(
    () =>
      events.filter(
        (event) =>
          `${event.ticker} ${event.company} ${event.headline}`
            .toLowerCase()
            .includes(search.toLowerCase()) &&
          (filter === "All events" || event.severity === filter) &&
          (!alertOnly || severityRank[event.severity] >= 3),
      ),
    [events, search, filter, alertOnly],
  );
  return (
    <section className="panel">
      <div className="panel-heading flex-wrap gap-4">
        <div>
          <div className="label text-primary">SIMULATED MONITORING</div>
          <h2 className="mt-2 text-lg font-semibold">
            {alertOnly ? "Priority alerts" : "AI risk signals"}
          </h2>
          <p className="mt-2 text-xs text-muted-foreground">
            Illustrative signals remain separate from live Alpaca account data.
          </p>
        </div>
        <div className="flex w-full flex-wrap gap-2 sm:w-auto">
          <div className="relative min-w-0 flex-1 sm:w-52">
            <Search size={15} className="absolute left-3 top-3 text-muted-foreground" />
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search signals"
              aria-label="Search signals"
              className="h-10 pl-9 text-xs"
            />
          </div>
          <div className="relative">
            <SlidersHorizontal
              size={14}
              className="pointer-events-none absolute left-3 top-3 text-muted-foreground"
            />
            <select
              value={filter}
              onChange={(event) => setFilter(event.target.value)}
              aria-label="Filter severity"
              className="h-10 rounded-md border border-border bg-secondary/40 pl-9 pr-3 text-xs"
            >
              <option>All events</option>
              <option>Critical</option>
              <option>High</option>
              <option>Moderate</option>
              <option>Low</option>
            </select>
          </div>
        </div>
      </div>
      <div className="mt-6 overflow-x-auto">
        <div className="feed-table min-w-[670px]">
          <div className="feed-row feed-head">
            <span>EVENT / SOURCE</span>
            <span>ASSET</span>
            <span>SENTIMENT</span>
            <span>IMPACT</span>
            <span>SEVERITY</span>
            <span />
          </div>
          {shown.map((event) => (
            <Button
              variant="ghost"
              key={event.id}
              className="feed-row feed-entry"
              onClick={() => onSelect(event)}
            >
              <span className="min-w-0 text-left">
                <span className="block truncate text-xs font-medium">{event.headline}</span>
                <span className="mt-2 block text-[10px] text-muted-foreground">
                  {event.source} · {event.time}
                </span>
              </span>
              <span className="text-left">
                <span className="block text-xs font-semibold">{event.ticker}</span>
                <span className="mt-1 block truncate text-[10px] text-muted-foreground">
                  {event.company}
                </span>
              </span>
              <span
                className={
                  event.sentiment < 0
                    ? "text-left text-xs text-destructive"
                    : "text-left text-xs text-primary"
                }
              >
                {event.sentiment > 0 ? "+" : ""}
                {event.sentiment.toFixed(2)}
              </span>
              <span className="text-left text-xs">{event.impact}/10</span>
              <span className={`severity severity-${event.severity.toLowerCase()}`}>
                {event.severity}
              </span>
              <ArrowRight size={14} />
            </Button>
          ))}
        </div>
      </div>
    </section>
  );
}

function DemoLocked() {
  return (
    <section className="panel flex min-h-80 flex-col items-center justify-center text-center">
      <ShieldAlert size={30} className="text-primary" />
      <h2 className="mt-4 text-xl font-semibold">Sign in for live account data</h2>
      <p className="mt-2 max-w-md text-sm text-muted-foreground">
        Portfolio, risk, market data, and paper trading require an authenticated workspace.
      </p>
      <Button asChild className="mt-5">
        <Link to="/auth">
          Sign in <ArrowRight size={14} />
        </Link>
      </Button>
    </section>
  );
}

function viewDescription(view: string) {
  switch (view) {
    case "Overview":
      return "Your invested stocks, focused market watchlist, sentiment context, and account-relevant news.";
    case "Live risk feed":
      return "Current position risk, calculation confidence, exposure, and review actions.";
    case "Portfolio status":
      return "Overall and individual risk calculated from real Alpaca paper-account positions.";
    case "Live Trading":
      return "Search tradable stocks, inspect live candlesticks, and submit confirmed paper orders.";
    case "Alerts":
      return "Priority simulated monitoring alerts.";
    default:
      return "Simulated AI monitoring signals for interface development.";
  }
}
