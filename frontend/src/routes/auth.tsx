import { useEffect, useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Activity, ArrowLeft, ArrowRight, ShieldCheck, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MarketScene } from "@/components/market/MarketScene";
import { fetchCurrentUser, googleSignInURL } from "@/lib/auth-api";

export const Route = createFileRoute("/auth")({
  validateSearch: (
    search: Record<string, unknown>,
  ): { mode?: "signup" | "login"; error?: string } => {
    const result: { mode?: "signup" | "login"; error?: string } = {};
    if (search["mode"] === "signup" || search["mode"] === "login") result.mode = search["mode"];
    if (typeof search["error"] === "string") result.error = search["error"];
    return result;
  },
  head: () => ({
    meta: [
      { title: "Continue with Google — Trading wens" },
      {
        name: "description",
        content: "Use your Google account to access the Trading wens risk workspace.",
      },
      { property: "og:title", content: "Access Trading wens" },
      { property: "og:description", content: "Secure Google access to Trading wens." },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: AuthPage,
});

function AuthPage() {
  const navigate = useNavigate();
  const { mode, error } = Route.useSearch();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    let active = true;
    fetchCurrentUser()
      .then((user) => {
        if (active && user) navigate({ to: "/dashboard", replace: true });
      })
      .catch(() => undefined)
      .finally(() => {
        if (active) setChecking(false);
      });
    return () => {
      active = false;
    };
  }, [navigate]);

  const signup = mode === "signup";
  return (
    <div className="auth-layout min-h-screen bg-background text-foreground">
      <div className="auth-form-side">
        <Link to="/" className="brand flex items-center gap-2.5">
          <span className="brand-mark">
            <Activity size={17} strokeWidth={2.5} />
          </span>
          <span>
            Trading <span className="text-primary">wens</span>
          </span>
        </Link>
        <div className="mx-auto flex w-full max-w-[410px] flex-1 flex-col justify-center py-16">
          <Link
            to="/"
            className="mb-10 flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft size={14} /> Back to home
          </Link>
          <div className="label text-primary">
            {signup ? "START YOUR WORKSPACE" : "WELCOME BACK"}
          </div>
          <h1 className="mt-3 text-3xl font-semibold sm:text-4xl">
            {signup ? "Create your workspace." : "Access your intelligence."}
          </h1>
          <p className="mt-3 text-sm leading-6 text-muted-foreground">
            Continue securely with your Google account. No separate Trading wens password is
            required.
          </p>

          {error && (
            <p
              role="alert"
              className="mt-7 rounded-md border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive"
            >
              {error}
            </p>
          )}

          {checking ? (
            <Button className="mt-9 h-12 w-full" disabled>
              Checking session…
            </Button>
          ) : (
            <Button asChild className="mt-9 h-12 w-full">
              <a href={googleSignInURL()} aria-label="Continue with Google">
                <span aria-hidden="true" className="text-base font-bold">
                  G
                </span>
                {signup ? "Sign up with Google" : "Continue with Google"}
                <ArrowRight size={16} />
              </a>
            </Button>
          )}
          <p className="mt-4 text-center text-[11px] leading-5 text-muted-foreground">
            Google shares your verified email, name, and profile image. Trading wens never receives
            your Google password.
          </p>
          <div className="mt-8 text-center text-sm text-muted-foreground">
            {signup ? "Already connected?" : "New to Trading wens?"}{" "}
            <Link
              to="/auth"
              search={{ mode: signup ? "login" : "signup" }}
              className="text-primary hover:underline"
            >
              {signup ? "Sign in" : "Create workspace"}
            </Link>
          </div>
          <div className="mt-12 flex items-center gap-2 text-xs text-muted-foreground">
            <ShieldCheck size={15} className="text-primary" /> Google OAuth with a secure, HTTP-only
            application session
          </div>
        </div>
        <div className="text-[11px] text-muted-foreground">© 2026 TRADING WENS</div>
      </div>
      <div className="auth-art relative hidden overflow-hidden lg:flex">
        <MarketScene />
        <div className="hero-vignette absolute inset-0" />
        <div className="relative z-10 mt-auto max-w-lg p-16">
          <span className="label flex items-center gap-2 text-primary">
            <Sparkles size={14} /> CLARITY IN COMPLEXITY
          </span>
          <h2 className="mt-6 text-4xl font-semibold leading-tight">
            Know the signal.
            <br />
            Before the market does.
          </h2>
          <p className="mt-5 text-sm leading-7 text-muted-foreground">
            Where information becomes perspective, and perspective becomes better decisions.
          </p>
          <div className="mt-10 flex items-center gap-3 border-t border-border pt-6 text-xs text-muted-foreground">
            <span className="live-pulse size-2 rounded-full bg-primary" /> Monitoring the pulse of
            the market
          </div>
        </div>
      </div>
    </div>
  );
}
