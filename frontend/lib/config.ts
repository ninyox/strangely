/**
 * The one place the frontend learns where the API lives.
 *
 * VITE_API_BASE_URL is the single documented setting (see .env.example). It
 * defaults to the local Go server so `bun run dev` works with no configuration
 * at all.
 */
// const DEFAULT_API_BASE_URL = "http://localhost:8080";
const DEFAULT_API_BASE_URL= "https://serve.subll.org"

function readEnv(key: string): string | undefined {
  const env = import.meta.env as Record<string, unknown> | undefined;
  const value = env?.[key];
  return typeof value === "string" && value.trim() !== "" ? value.trim() : undefined;
}

export const API_BASE_URL = (readEnv("VITE_API_BASE_URL") ?? DEFAULT_API_BASE_URL).replace(/\/+$/, "");

/** The API root, e.g. http://localhost:8080/api */
export const API_ROOT = `${API_BASE_URL}/api`;

/**
 * The signaling WebSocket URL, derived from the API base so there is only ever
 * one setting to change: http becomes ws, https becomes wss.
 */
export function signalingUrl(): string {
  const base = new URL(`${API_BASE_URL}/api/signaling`, window.location.origin);
  base.protocol = base.protocol === "https:" ? "wss:" : "ws:";
  return base.toString();
}

/** Where the visitor can read about the project. */
export const ROUTES = {
  home: "/",
  chat: "/chat",
  privacy: "/privacy",
  guidelines: "/guidelines",
  about: "/about",
} as const;

/** Stored keys. Only the display name is kept locally, never a credential. */
export const STORAGE_KEYS = {
  username: "strangerly.username",
} as const;
