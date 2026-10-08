import { z } from "zod";

const googleUserSchema = z.object({
  id: z.string(),
  email: z.string().email(),
  display_name: z.string().optional(),
  avatar_url: z.string().url().optional(),
});

const sessionSchema = z.object({ user: googleUserSchema });

export type GoogleUser = z.infer<typeof googleUserSchema>;

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

export async function fetchCurrentUser(): Promise<GoogleUser | null> {
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
    throw new Error("Your Google session has expired. Sign in again.");
  }
  return response;
}
