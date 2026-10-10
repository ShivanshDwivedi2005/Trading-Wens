import { FormEvent, useEffect, useState } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Activity, ArrowLeft, ArrowRight, ShieldCheck, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { MarketScene } from "@/components/market/MarketScene";
import {
  fetchAuthProviders,
  fetchCurrentUser,
  googleSignInURL,
  signIn,
  signUp,
} from "@/lib/auth-api";

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
      { title: "Sign in — Trading wens" },
      { name: "description", content: "Access the Trading wens risk workspace securely." },
      { property: "og:title", content: "Access Trading wens" },
      { property: "og:description", content: "Secure access to Trading wens." },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: AuthPage,
});

function AuthPage() {
  const navigate = useNavigate();
  const { mode, error } = Route.useSearch();
  const signup = mode === "signup";
  const [checking, setChecking] = useState(true);
  const [googleEnabled, setGoogleEnabled] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState("");
  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [identity, setIdentity] = useState("");
  const [password, setPassword] = useState("");

  useEffect(() => {
    let active = true;
    Promise.all([fetchCurrentUser(), fetchAuthProviders()])
      .then(([user, providers]) => {
        if (!active) return;
        setGoogleEnabled(providers.google);
        if (user) navigate({ to: "/dashboard", replace: true });
      })
      .catch(() => undefined)
      .finally(() => active && setChecking(false));
    return () => {
      active = false;
    };
  }, [navigate]);

  useEffect(() => {
    setFormError("");
    setPassword("");
  }, [signup]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setFormError("");
    try {
      if (signup) {
        await signUp({ username, email, password, display_name: displayName });
      } else {
        await signIn(identity, password);
      }
      await navigate({ to: "/dashboard", replace: true });
    } catch (submissionError) {
      setFormError(
        submissionError instanceof Error ? submissionError.message : "Authentication failed.",
      );
    } finally {
      setSubmitting(false);
    }
  }

  const visibleError = formError || error;
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
        <div className="mx-auto flex w-full max-w-[410px] flex-1 flex-col justify-center py-12">
          <Link
            to="/"
            className="mb-8 flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground"
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
            {signup
              ? "Create an account with email and password, or use Google."
              : "Sign in with your email or username, or continue with Google."}
          </p>

          {visibleError && (
            <p
              role="alert"
              className="mt-5 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive"
            >
              {visibleError}
            </p>
          )}

          <form className="mt-6 space-y-4" onSubmit={submit} noValidate>
            {signup ? (
              <>
                <AuthField
                  id="display-name"
                  label="Display name"
                  value={displayName}
                  onChange={setDisplayName}
                  autoComplete="name"
                />
                <AuthField
                  id="username"
                  label="Username"
                  value={username}
                  onChange={setUsername}
                  autoComplete="username"
                  minLength={3}
                  maxLength={32}
                  pattern="[A-Za-z0-9_.-]+"
                  required
                  hint="3–32 letters, numbers, dots, dashes, or underscores."
                />
                <AuthField
                  id="email"
                  label="Email"
                  type="email"
                  value={email}
                  onChange={setEmail}
                  autoComplete="email"
                  required
                />
              </>
            ) : (
              <AuthField
                id="identity"
                label="Email or username"
                value={identity}
                onChange={setIdentity}
                autoComplete="username"
                required
              />
            )}
            <AuthField
              id="password"
              label="Password"
              type="password"
              value={password}
              onChange={setPassword}
              autoComplete={signup ? "new-password" : "current-password"}
              minLength={signup ? 12 : undefined}
              maxLength={72}
              required
              hint={signup ? "Use at least 12 characters." : undefined}
            />
            <Button className="h-11 w-full" type="submit" disabled={checking || submitting}>
              {submitting ? "Please wait…" : signup ? "Create account" : "Sign in"}
              {!submitting && <ArrowRight size={16} />}
            </Button>
          </form>

          <div
            className="my-5 flex items-center gap-3 text-[11px] text-muted-foreground"
            aria-hidden="true"
          >
            <span className="h-px flex-1 bg-border" />
            <span>OR</span>
            <span className="h-px flex-1 bg-border" />
          </div>
          {googleEnabled ? (
            <Button asChild variant="outline" className="h-11 w-full">
              <a href={googleSignInURL()}>
                <span aria-hidden="true" className="font-bold">
                  G
                </span>
                {signup ? "Sign up with Google" : "Continue with Google"}
              </a>
            </Button>
          ) : (
            <Button variant="outline" className="h-11 w-full" disabled>
              Google sign-in is not configured
            </Button>
          )}
          <p className="mt-3 text-center text-[11px] leading-5 text-muted-foreground">
            Your password is hashed before storage and is never returned to the browser.
          </p>
          <div className="mt-6 text-center text-sm text-muted-foreground">
            {signup ? "Already have an account?" : "New to Trading wens?"}{" "}
            <Link
              to="/auth"
              search={{ mode: signup ? "login" : "signup" }}
              className="text-primary hover:underline"
            >
              {signup ? "Sign in" : "Create account"}
            </Link>
          </div>
          <div className="mt-8 flex items-center gap-2 text-xs text-muted-foreground">
            <ShieldCheck size={15} className="text-primary" /> Secure, HTTP-only application session
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

type AuthFieldProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  autoComplete: string;
  required?: boolean;
  minLength?: number | undefined;
  maxLength?: number;
  pattern?: string;
  hint?: string | undefined;
};

function AuthField({ id, label, value, onChange, hint, ...props }: AuthFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-11"
        aria-describedby={hint ? `${id}-hint` : undefined}
        {...props}
      />
      {hint && (
        <p id={`${id}-hint`} className="text-[11px] text-muted-foreground">
          {hint}
        </p>
      )}
    </div>
  );
}
