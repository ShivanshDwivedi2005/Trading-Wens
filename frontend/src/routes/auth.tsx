import { useEffect, useState, type FormEvent } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import {
  Activity,
  ArrowLeft,
  ArrowRight,
  Eye,
  EyeOff,
  LockKeyhole,
  Mail,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { MarketScene } from "@/components/market/MarketScene";
import { supabase } from "@/integrations/supabase/client";

export const Route = createFileRoute("/auth")({
  validateSearch: (search: Record<string, unknown>): { mode?: "signup" | "forgot" | "login" } => ({
    mode: search["mode"] === "signup" || search["mode"] === "forgot" ? search["mode"] : "login",
  }),
  head: () => ({
    meta: [
      { title: "Sign in or create an account — Trading wens" },
      {
        name: "description",
        content:
          "Access your financial risk intelligence workspace or create a new Trading wens account.",
      },
      { property: "og:title", content: "Access Trading wens" },
      {
        property: "og:description",
        content: "Sign in or create an account to access your financial risk workspace.",
      },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: AuthPage,
});

function AuthPage() {
  const navigate = useNavigate();
  const { mode: initialMode } = Route.useSearch();
  const [mode, setMode] = useState<"login" | "signup" | "forgot">(initialMode ?? "login");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  useEffect(() => {
    setMode(initialMode ?? "login");
  }, [initialMode]);
  useEffect(() => {
    supabase.auth.getUser().then(({ data }) => {
      if (data.user) navigate({ to: "/dashboard", replace: true });
    });
  }, [navigate]);
  const changeMode = (next: "login" | "signup" | "forgot") => {
    setMode(next);
    setError("");
    setNotice("");
    navigate({ to: "/auth", search: { mode: next }, replace: true });
  };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setNotice("");
    const parsed = z.string().email().safeParse(email);
    if (!parsed.success) {
      setError("Enter a valid email address.");
      return;
    }
    if (mode === "signup" && name.trim().length < 2) {
      setError("Enter your name to continue.");
      return;
    }
    if (mode !== "forgot" && password.length < 8) {
      setError("Use a password with at least 8 characters.");
      return;
    }
    setBusy(true);
    try {
      if (mode === "forgot") {
        const { error } = await supabase.auth.resetPasswordForEmail(email, {
          redirectTo: `${window.location.origin}/reset-password`,
        });
        if (error) throw error;
        setNotice("If an account exists for that address, a reset link is on its way.");
      } else if (mode === "signup") {
        const { data, error } = await supabase.auth.signUp({
          email,
          password,
          options: { emailRedirectTo: window.location.origin, data: { full_name: name.trim() } },
        });
        if (error) throw error;
        if (data.session && data.user) {
          await supabase.from("profiles").upsert({ id: data.user.id, display_name: name.trim() });
          navigate({ to: "/dashboard" });
        } else setNotice("Check your email to confirm your account, then come back to sign in.");
      } else {
        const { error } = await supabase.auth.signInWithPassword({ email, password });
        if (error) throw error;
        navigate({ to: "/dashboard" });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Something went wrong. Please try again.");
    } finally {
      setBusy(false);
    }
  };
  const google = async () => {
    setBusy(true);
    setError("");
    try {
      const { data, error } = await supabase.auth.signInWithOAuth({
        provider: "google",
        options: { redirectTo: `${window.location.origin}/dashboard` },
      });
      if (error) throw error;
      if (!data.url) throw new Error("Google sign-in did not return a redirect URL.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Google sign-in failed.");
    } finally {
      setBusy(false);
    }
  };
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
            {mode === "signup"
              ? "START YOUR WORKSPACE"
              : mode === "forgot"
                ? "ACCOUNT RECOVERY"
                : "WELCOME BACK"}
          </div>
          <h1 className="mt-3 text-3xl font-semibold sm:text-4xl">
            {mode === "signup"
              ? "Create your account."
              : mode === "forgot"
                ? "Reset your password."
                : "Access your intelligence."}
          </h1>
          <p className="mt-3 text-sm leading-6 text-muted-foreground">
            {mode === "signup"
              ? "A more informed view of market risk starts here."
              : mode === "forgot"
                ? "We’ll send a secure reset link to your email."
                : "Your market intelligence workspace is ready."}
          </p>
          {notice ? (
            <div
              role="status"
              className="mt-9 rounded-md border border-primary/30 bg-primary/10 p-5 text-sm leading-6 text-primary"
            >
              {notice}
              <Button
                variant="link"
                className="mt-3 block h-auto p-0 text-primary"
                onClick={() => changeMode("login")}
              >
                Return to login <ArrowRight size={14} />
              </Button>
            </div>
          ) : (
            <>
              <form className="mt-9 space-y-5" onSubmit={submit}>
                {mode === "signup" && (
                  <label className="block text-xs font-medium">
                    Full name
                    <Input
                      autoComplete="name"
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder="Your name"
                      className="mt-2 h-11 border-border bg-secondary/50"
                      required
                    />
                  </label>
                )}
                <label className="block text-xs font-medium">
                  Work email
                  <div className="relative mt-2">
                    <Mail size={16} className="absolute left-3 top-3.5 text-muted-foreground" />
                    <Input
                      type="email"
                      autoComplete="email"
                      value={email}
                      onChange={(e) => setEmail(e.target.value)}
                      placeholder="you@company.com"
                      className="h-11 border-border bg-secondary/50 pl-10"
                      required
                    />
                  </div>
                </label>
                {mode !== "forgot" && (
                  <label className="block text-xs font-medium">
                    Password
                    <div className="relative mt-2">
                      <LockKeyhole
                        size={16}
                        className="absolute left-3 top-3.5 text-muted-foreground"
                      />
                      <Input
                        type={showPassword ? "text" : "password"}
                        autoComplete={mode === "signup" ? "new-password" : "current-password"}
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder="At least 8 characters"
                        className="h-11 border-border bg-secondary/50 pl-10 pr-10"
                        required
                      />
                      <Button
                        type="button"
                        size="icon"
                        variant="ghost"
                        aria-label={showPassword ? "Hide password" : "Show password"}
                        className="absolute right-1 top-1"
                        onClick={() => setShowPassword(!showPassword)}
                      >
                        {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                      </Button>
                    </div>
                  </label>
                )}
                {mode === "login" && (
                  <div className="text-right">
                    <Button
                      type="button"
                      variant="link"
                      className="h-auto p-0 text-xs text-muted-foreground"
                      onClick={() => changeMode("forgot")}
                    >
                      Forgot password?
                    </Button>
                  </div>
                )}
                {error && (
                  <p role="alert" className="text-sm text-destructive">
                    {error}
                  </p>
                )}
                <Button type="submit" className="h-11 w-full" disabled={busy}>
                  {busy
                    ? "Please wait…"
                    : mode === "signup"
                      ? "Create account"
                      : mode === "forgot"
                        ? "Send reset link"
                        : "Login"}{" "}
                  <ArrowRight size={16} />
                </Button>
              </form>
              {mode !== "forgot" && (
                <>
                  <div className="my-7 flex items-center gap-4 text-[10px] tracking-wider text-muted-foreground">
                    <div className="h-px flex-1 bg-border" /> OR CONTINUE WITH{" "}
                    <div className="h-px flex-1 bg-border" />
                  </div>
                  <Button
                    variant="outline"
                    className="h-11 w-full border-border bg-secondary/30"
                    onClick={google}
                    disabled={busy}
                  >
                    <span className="text-base font-bold">G</span> Google
                  </Button>
                </>
              )}
              <div className="mt-8 text-center text-sm text-muted-foreground">
                {mode === "signup"
                  ? "Already have an account?"
                  : mode === "forgot"
                    ? "Remember your password?"
                    : "New to Trading wens?"}{" "}
                <Button
                  variant="link"
                  className="h-auto p-0 text-sm text-primary"
                  onClick={() => changeMode(mode === "login" ? "signup" : "login")}
                >
                  {mode === "login" ? "Create account" : "Sign in"}
                </Button>
              </div>
            </>
          )}
          <div className="mt-12 flex items-center gap-2 text-xs text-muted-foreground">
            <ShieldCheck size={15} className="text-primary" /> Secure access to your intelligence
            workspace
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
