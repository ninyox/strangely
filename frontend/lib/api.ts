import axios, { type AxiosResponse } from "axios";

import { API_ROOT } from "./config";
import type {
  ApiEnvelope,
  EndPayload,
  PublicConfig,
  ReportPayload,
  SessionPayload,
  StatePayload,
} from "./types";

/**
 * ApiError is the single failure shape the UI handles. `code` is the stable,
 * machine readable reason the backend sends (skip_locked, already_queued,
 * rate_limited, …), which is what lets the call screen react to a refused Skip
 * without matching on English copy.
 */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string | undefined,
    message: string,
    readonly data?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

const client = axios.create({
  baseURL: API_ROOT,
  // The anonymous session lives in an HttpOnly cookie, so every request has to
  // carry credentials. The token itself is never visible to this code.
  withCredentials: true,
  timeout: 15000,
  headers: { "Content-Type": "application/json" },
});

function toApiError(error: unknown): ApiError {
  if (axios.isAxiosError(error)) {
    const response = error.response;
    if (response) {
      const envelope = response.data as ApiEnvelope<unknown> | undefined;
      return new ApiError(
        response.status,
        envelope?.code,
        envelope?.message ?? error.message,
        envelope?.data,
      );
    }
    return new ApiError(
      0,
      "network_error",
      "Could not reach Strangerly. Check your connection and try again.",
    );
  }

  if (error instanceof Error) {
    return new ApiError(0, "unknown_error", error.message);
  }

  return new ApiError(0, "unknown_error", "Something went wrong. Please try again.");
}

async function unwrap<T>(request: Promise<AxiosResponse<ApiEnvelope<T>>>): Promise<T> {
  try {
    const response = await request;
    return response.data.data as T;
  } catch (error) {
    throw toApiError(error);
  }
}

export const api = {
  /** Create or resume the anonymous visit. */
  createSession: (username?: string) =>
    unwrap<SessionPayload>(client.post<ApiEnvelope<SessionPayload>>("/session", username ? { username } : {})),

  /** Ask whether the cookie is still good, without creating anything. */
  getSession: () => unwrap<SessionPayload>(client.get<ApiEnvelope<SessionPayload>>("/session")),

  /** ICE servers, the Skip window, and the accepted report reasons. */
  getConfig: () => unwrap<PublicConfig>(client.get<ApiEnvelope<PublicConfig>>("/config")),

  join: (username?: string) =>
    unwrap<StatePayload>(client.post<ApiEnvelope<StatePayload>>("/matchmaking/join", username ? { username } : {})),

  cancel: () => unwrap<StatePayload>(client.post<ApiEnvelope<StatePayload>>("/matchmaking/cancel", {})),

  status: () => unwrap<StatePayload>(client.get<ApiEnvelope<StatePayload>>("/matchmaking/status")),

  skip: (matchId: string) =>
    unwrap<EndPayload>(client.post<ApiEnvelope<EndPayload>>(`/matches/${encodeURIComponent(matchId)}/skip`, {})),

  stop: (matchId: string) =>
    unwrap<EndPayload>(client.post<ApiEnvelope<EndPayload>>(`/matches/${encodeURIComponent(matchId)}/stop`, {})),

  report: (matchId: string, reason: string, note?: string) =>
    unwrap<ReportPayload>(
      client.post<ApiEnvelope<ReportPayload>>(`/matches/${encodeURIComponent(matchId)}/report`, { reason, note }),
    ),
};

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

export function apiErrorMessage(error: unknown, fallback = "Something went wrong. Please try again."): string {
  if (isApiError(error)) {
    return error.message || fallback;
  }
  if (error instanceof Error) {
    return error.message || fallback;
  }
  return fallback;
}

export function apiErrorCode(error: unknown): string | undefined {
  return isApiError(error) ? error.code : undefined;
}

/** The remaining milliseconds a refused Skip reported, if any. */
export function skipLockedRemainingMs(error: unknown): number | undefined {
  if (!isApiError(error) || error.code !== "skip_locked") {
    return undefined;
  }

  const data = error.data as { remainingMs?: number } | undefined;
  return typeof data?.remainingMs === "number" ? data.remainingMs : undefined;
}

export default api;
