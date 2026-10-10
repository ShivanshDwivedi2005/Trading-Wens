import { z } from "zod";

const authUserSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  display_name: z.string().optional(),
  avatar_url: z.string().url().optional(),
});

const sessionSchema = z.object({ user: authUserSchema });
const providersSchema = z.object({ password: z.boolean(), google: z.boolean() });

export type AuthUser = z.infer<typeof authUserSchema>;

export function parseAuthSession(value: unknown) {
  return sessionSchema.parse(value);
}

function apiOrigin() {
  return (import.meta.env["VITE_API_BASE_URL"]?.trim() ?? "")
    .replace(/\/$/, "")
    .replace(/\/api\/v1$/, "");
}

export function apiURL(path: string) {
  return `${apiOrigin()}${path}`;
}

export function googleSignInURL() {
  return apiURL("/api/v1/auth/google/start");
}

export async function fetchAuthProviders() {
  const response = await fetch(apiURL("/api/v1/auth/providers"), {
    credentials: "include",
    headers: { Accept: "application/json" },
  });
  if (!response.ok) return { password: true, google: false };
  return providersSchema.parse(await response.json());
}

type SignupDetails = {
  username: string;
  email: string;
  password: string;
  display_name: string;
};

async function credentialRequest(path: string, body: object): Promise<AuthUser> {
  const response = await fetch(apiURL(path), {
    method: "POST",
    credentials: "include",
    headers: { Accept: "application/json", "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const message = z.object({ error: z.object({ message: z.string() }) }).safeParse(payload);
    throw new Error(message.success ? message.data.error.message : "Authentication failed.");
  }
  return parseAuthSession(payload).user;
}

export function signUp(details: SignupDetails) {
  return credentialRequest("/api/v1/auth/signup", details);
}

export function signIn(identity: string, password: string) {
  return credentialRequest("/api/v1/auth/login", { identity, password });
}

export async function fetchCurrentUser(): Promise<AuthUser | null> {
  const response = await fetch(apiURL("/api/v1/auth/session"), {
    credentials: "include",
    headers: { Accept: "application/json" },
  });
  if (response.status === 401) return null;
  if (!response.ok) throw new Error("Authentication is temporarily unavailable.");
  return parseAuthSession(await response.json()).user;
}

export async function signOut() {
  const response = await fetch(apiURL("/api/v1/auth/logout"), {
    method: "POST",
    credentials: "include",
    headers: { Accept: "application/json" },
  });
  if (!response.ok) throw new Error("Sign out could not be completed.");
}

export async function authenticatedFetch(path: string, init?: RequestInit) {
  const response = await fetch(apiURL(path), {
    ...init,
    credentials: "include",
    headers: { Accept: "application/json", ...init?.headers },
  });
  if (response.status === 401) {
    throw new Error("Your session has expired. Sign in again.");
  }
  return response;
}
