import { useEffect, useState, type FormEvent } from "react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { Activity, ArrowRight, LockKeyhole } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { supabase } from "@/integrations/supabase/client";
export const Route = createFileRoute("/reset-password")({
  head: () => ({
    meta: [
      { title: "Reset password — Trading wens" },
      { name: "description", content: "Set a new password for your Trading wens account." },
      { property: "og:title", content: "Reset password — Trading wens" },
      {
        property: "og:description",
        content: "Set a new password and return to your Trading wens workspace.",
      },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  }),
  component: ResetPassword,
});
function ResetPassword() {
  const navigate = useNavigate();
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [valid, setValid] = useState(false);
  useEffect(() => {
    const hash = new URLSearchParams(window.location.hash.slice(1));
    setValid(hash.get("type") === "recovery");
    const { data } = supabase.auth.onAuthStateChange((event) => {
      if (event === "PASSWORD_RECOVERY") setValid(true);
    });
    return () => data.subscription.unsubscribe();
  }, []);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (password.length < 8) {
      setError("Use at least 8 characters.");
      return;
    }
    if (password !== confirm) {
      setError("Passwords do not match.");
      return;
    }
    setBusy(true);
    setError("");
    const { error } = await supabase.auth.updateUser({ password });
    setBusy(false);
    if (error) setError(error.message);
    else navigate({ to: "/dashboard", replace: true });
  };
  return (
    <div className="flex min-h-screen flex-col bg-background p-6 text-foreground sm:p-10">
      <Link to="/" className="brand flex items-center gap-2.5">
        <span className="brand-mark">
          <Activity size={17} />
        </span>
        <span>
          Trading <span className="text-primary">wens</span>
        </span>
      </Link>
      <div className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center">
        <div className="mb-5 flex size-12 items-center justify-center rounded-md bg-primary/10 text-primary">
          <LockKeyhole size={22} />
        </div>
        <h1 className="text-3xl font-semibold">Set a new password.</h1>
        <p className="mt-3 text-sm text-muted-foreground">
          Choose a secure password for your account.
        </p>
        {valid ? (
          <form onSubmit={submit} className="mt-8 space-y-4">
            <label className="block text-xs">
              New password
              <Input
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="mt-2 h-11"
                required
              />
            </label>
            <label className="block text-xs">
              Confirm password
              <Input
                type="password"
                autoComplete="new-password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                className="mt-2 h-11"
                required
              />
            </label>
            {error && (
              <p role="alert" className="text-sm text-destructive">
                {error}
              </p>
            )}
            <Button className="h-11 w-full" disabled={busy}>
              Update password <ArrowRight size={16} />
            </Button>
          </form>
        ) : (
          <div className="mt-8 rounded-md border border-border bg-secondary/40 p-5 text-sm text-muted-foreground">
            Open the reset link from your email to continue.{" "}
            <Link to="/auth" search={{ mode: "forgot" }} className="text-primary underline">
              Request another link
            </Link>
            .
          </div>
        )}
      </div>
    </div>
  );
}
