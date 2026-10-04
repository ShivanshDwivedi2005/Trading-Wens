import { useEffect, useState } from "react";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  Activity,
  ArrowRight,
  ArrowUpRight,
  BrainCircuit,
  ChevronRight,
  Crosshair,
  LockKeyhole,
  Menu,
  Radar,
  ShieldCheck,
  Sparkles,
  TrendingUp,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { MarketScene } from "@/components/market/MarketScene";
import { supabase } from "@/integrations/supabase/client";

export const Route = createFileRoute("/")({
  head: () => ({
    meta: [
      { title: "Trading wens — AI-Powered Financial Risk Intelligence" },
      {
        name: "description",
        content:
          "Turn real-time news and market conversations into actionable financial risk signals. Explore Trading wens with a public simulated demo.",
      },
      { property: "og:title", content: "Trading wens — Financial Risk Intelligence" },
      {
        property: "og:description",
        content:
          "Turn real-time news and market conversations into actionable financial risk signals.",
      },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: Home,
});

function Home() {
  const [menuOpen, setMenuOpen] = useState(false);
  const [signedIn, setSignedIn] = useState(false);
  useEffect(() => {
    const onUser = (user: boolean) => setSignedIn(user);
    supabase.auth.getUser().then(({ data }) => onUser(Boolean(data.user)));
    const { data: listener } = supabase.auth.onAuthStateChange((_event, session) =>
      onUser(Boolean(session?.user)),
    );
    return () => listener.subscription.unsubscribe();
  }, []);
  return (
    <div className="landing min-h-screen bg-background text-foreground">
      <header className="landing-header">
        <Link to="/" className="brand flex items-center gap-2.5">
          <span className="brand-mark">
            <Activity size={17} strokeWidth={2.5} />
          </span>
          <span>
            Trading <span className="text-primary">wens</span>
          </span>
        </Link>
        <div className="hidden items-center gap-8 md:flex">
          <span className="flex items-center gap-2 text-[11px] font-medium tracking-wide text-muted-foreground">
            <span className="live-pulse size-1.5 rounded-full bg-primary" /> SIGNAL SIMULATION
          </span>
          <Button
            asChild
            variant="ghost"
            className="text-sm text-muted-foreground hover:text-foreground"
          >
            <Link to="/demo">Explore demo</Link>
          </Button>
          <Button
            asChild
            variant="ghost"
            className="text-sm text-muted-foreground hover:text-foreground"
          >
            <Link to={signedIn ? "/dashboard" : "/auth"}>{signedIn ? "Dashboard" : "Login"}</Link>
          </Button>
          <Button asChild size="sm" className="h-9 px-5">
            {signedIn ? (
              <Link to="/dashboard">
                Open workspace <ArrowUpRight size={14} />
              </Link>
            ) : (
              <Link to="/auth" search={{ mode: "signup" }}>
                Get access <ArrowUpRight size={14} />
              </Link>
            )}
          </Button>
        </div>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Toggle navigation"
          className="md:hidden"
          onClick={() => setMenuOpen(!menuOpen)}
        >
          {menuOpen ? <X /> : <Menu />}
        </Button>
      </header>
      {menuOpen && (
        <nav className="mobile-landing-nav">
          <Link to="/demo" onClick={() => setMenuOpen(false)}>
            Explore demo
          </Link>
          <Link to="/auth" onClick={() => setMenuOpen(false)}>
            Login
          </Link>
          <Link to="/auth" search={{ mode: "signup" }} onClick={() => setMenuOpen(false)}>
            Create account
          </Link>
        </nav>
      )}
      <main>
        <section className="hero relative isolate flex min-h-[680px] items-center overflow-hidden">
          <MarketScene />
          <div className="hero-vignette absolute inset-0" />
          <div className="hero-ticker ticker-one">
            <span className="text-primary">●</span> NVDA SENTIMENT{" "}
            <span className="text-primary">+0.82</span>
          </div>
          <div className="hero-ticker ticker-two">
            <span className="text-destructive">●</span> GEOPOLITICAL RISK DETECTED
          </div>
          <div className="hero-ticker ticker-three">
            PORTFOLIO EXPOSURE UPDATED <span className="text-primary">↗</span>
          </div>
          <div className="hero-ticker ticker-four">
            HIGH IMPACT EVENT DETECTED <span className="text-accent">◆</span>
          </div>
          <div className="hero-ticker ticker-five">
            MARKET RISK REPORT <span className="text-primary">/ READY</span>
          </div>
          <div className="hero-content relative z-10 mx-auto w-full max-w-7xl px-6 pb-28 pt-20 sm:px-10 lg:px-16">
            <div className="hero-kicker mb-8 inline-flex items-center gap-3 rounded-sm border border-primary/30 bg-primary/10 px-3 py-2 text-[10px] font-semibold tracking-[0.15em] text-primary">
              <span className="live-pulse size-1.5 rounded-full bg-primary" /> THE SIGNAL BEHIND THE
              NOISE <ChevronRight size={13} />
            </div>
            <h1 className="max-w-[850px] text-5xl font-semibold leading-[1.1] sm:text-6xl lg:text-[82px]">
              AI-Powered
              <br />
              Financial Risk
              <br />
              <span className="text-primary">Intelligence.</span>
            </h1>
            <p className="mt-7 max-w-xl text-base leading-8 text-muted-foreground sm:text-lg">
              Turn real-time news and market conversations into actionable financial risk signals.
            </p>
            <div className="mt-10 flex flex-wrap items-center gap-3">
              <Button asChild size="lg" className="h-12 px-7 text-sm">
                <Link to="/demo">
                  Try Live Demo <ArrowUpRight size={17} />
                </Link>
              </Button>
              <Button
                asChild
                size="lg"
                variant="outline"
                className="h-12 border-foreground/25 bg-background/30 px-7 text-sm backdrop-blur-md"
              >
                <Link to="/auth" search={{ mode: "signup" }}>
                  Create Account <ArrowRight size={16} />
                </Link>
              </Button>
              <Button
                asChild
                variant="ghost"
                size="lg"
                className="h-12 px-5 text-sm text-muted-foreground"
              >
                <Link to="/auth">Login</Link>
              </Button>
            </div>
            <div className="mt-16 flex flex-wrap items-center gap-x-8 gap-y-4 text-[10px] font-medium tracking-[0.11em] text-muted-foreground">
              <span className="flex items-center gap-2">
                <Radar size={14} className="text-primary" /> CONTINUOUS MONITORING
              </span>
              <span className="flex items-center gap-2">
                <ShieldCheck size={14} className="text-primary" /> EXPLAINABLE SIGNALS
              </span>
              <span className="flex items-center gap-2">
                <LockKeyhole size={14} className="text-primary" /> BUILT FOR DECISIONS
              </span>
            </div>
          </div>
          <div className="hero-bottom absolute bottom-0 z-10 w-full border-t border-border/60 bg-background/40 backdrop-blur-sm">
            <div className="mx-auto flex max-w-7xl items-center justify-between gap-6 overflow-hidden px-6 py-4 text-[10px] tracking-[0.08em] text-muted-foreground sm:px-10 lg:px-16">
              <span className="shrink-0 text-primary">● LIVE MARKET PULSE</span>
              <span className="shrink-0">
                S&P 500 <b className="ml-2 font-medium text-foreground">5,842.47</b>{" "}
                <b className="ml-1 font-medium text-primary">+0.84%</b>
              </span>
              <span className="shrink-0">
                NASDAQ <b className="ml-2 font-medium text-foreground">18,394.21</b>{" "}
                <b className="ml-1 font-medium text-primary">+1.26%</b>
              </span>
              <span className="shrink-0">
                VIX <b className="ml-2 font-medium text-foreground">17.34</b>{" "}
                <b className="ml-1 font-medium text-destructive">+2.41%</b>
              </span>
              <span className="hidden shrink-0 lg:inline">DATA FOR ILLUSTRATION</span>
            </div>
          </div>
        </section>
        <section className="border-b border-border bg-secondary/25">
          <div className="mx-auto grid max-w-7xl gap-10 px-6 py-20 sm:px-10 lg:grid-cols-[.9fr_1.1fr] lg:gap-24 lg:px-16 lg:py-28">
            <div>
              <div className="label text-primary">FROM INFORMATION TO INTELLIGENCE</div>
              <h2 className="mt-5 max-w-lg text-4xl font-semibold leading-tight sm:text-5xl">
                Markets move fast.
                <br />
                <span className="text-muted-foreground">Know why.</span>
              </h2>
              <p className="mt-6 max-w-md text-sm leading-7 text-muted-foreground">
                Trading wens connects the dots between breaking stories, shifting sentiment, and the
                positions that matter. Every signal comes with the context behind it.
              </p>
              <Button asChild variant="link" className="mt-6 h-auto p-0 text-sm">
                <Link to="/demo">
                  Explore the live demo <ArrowUpRight size={16} />
                </Link>
              </Button>
            </div>
            <div className="grid gap-px border border-border bg-border sm:grid-cols-2">
              {[
                {
                  icon: Radar,
                  number: "01",
                  title: "Hear the market",
                  description: "Follow news and social conversations as market narratives change.",
                },
                {
                  icon: BrainCircuit,
                  number: "02",
                  title: "Understand the signal",
                  description:
                    "See sentiment, event classification, impact, and confidence in one view.",
                },
                {
                  icon: Crosshair,
                  number: "03",
                  title: "Find your exposure",
                  description: "Connect emerging events to the positions in your portfolio.",
                },
                {
                  icon: TrendingUp,
                  number: "04",
                  title: "Prepare for what’s next",
                  description: "Explore potential stress scenarios before they become surprises.",
                },
              ].map((item) => (
                <div
                  key={item.number}
                  className="bg-background p-7 transition-colors hover:bg-secondary/50"
                >
                  <div className="flex items-center justify-between">
                    <item.icon size={22} className="text-primary" />
                    <span className="label text-muted-foreground">{item.number}</span>
                  </div>
                  <h3 className="mt-8 text-lg font-semibold">{item.title}</h3>
                  <p className="mt-3 text-sm leading-6 text-muted-foreground">{item.description}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
        <section className="border-b border-border bg-background">
          <div className="mx-auto flex max-w-7xl flex-col items-start justify-between gap-8 px-6 py-20 sm:px-10 lg:flex-row lg:items-center lg:px-16">
            <div>
              <div className="label text-primary">SEE THE INTELLIGENCE IN ACTION</div>
              <h2 className="mt-4 text-3xl font-semibold sm:text-4xl">
                From headline to risk signal.
              </h2>
              <p className="mt-4 max-w-xl text-sm leading-7 text-muted-foreground">
                Open an event, inspect the reasoning, and see how it translates into a portfolio
                perspective.
              </p>
            </div>
            <Button asChild size="lg" className="h-12 px-7">
              <Link to="/demo">
                Launch public demo <ArrowUpRight size={16} />
              </Link>
            </Button>
          </div>
        </section>
      </main>
      <footer className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-6 py-8 text-xs text-muted-foreground sm:px-10 lg:px-16">
        <span>© 2026 TRADING WENS</span>
        <span>Market intelligence, with context.</span>
      </footer>
    </div>
  );
}
