/**
 * Shared API and call types.
 *
 * These mirror the Go payloads exactly. The server is the authority for every
 * value here; nothing in this file invents state the backend does not send.
 */

/** The server's lifecycle value for a session. */
export type MatchState = "idle" | "queued" | "connecting" | "matched" | "ended";

/** The screen the visitor is looking at. `connecting` covers the moment
 * between "matched" and "the peer connection is up". */
export type CallPhase = "idle" | "queued" | "connecting" | "connected" | "ended";

/** Why a call ended, as reported by the server. */
export type EndReason = "skipped" | "stopped" | "disconnected" | "left" | "session_ended" | "too_long";

export type SessionPayload = {
  sessionId: string;
  username?: string;
  state: MatchState;
  skipLockMs: number;
  expiresInSeconds: number;
  serverTime: string;
};

export type StatePayload = {
  state: MatchState;
  matchId?: string;
  partnerUsername?: string;
  initiator: boolean;
  matchedAt?: string;
  skipLockMs: number;
  remainingMs: number;
  queuePosition?: number;
  ready: boolean;
  peerReady: boolean;
  lastEndedReason?: EndReason;
  lastEndedMatchId?: string;
  serverTime: string;
};

export type EndPayload = {
  state: MatchState;
  matchId: string;
  reason: EndReason;
  alreadyEnded: boolean;
  serverTime: string;
};

export type ReportPayload = {
  reportId: string;
  matchId: string;
  receivedAt: string;
};

export type IceServer = {
  urls: string | string[];
  username?: string;
  credential?: string;
};

export type PublicConfig = {
  iceServers: IceServer[];
  skipLockSeconds: number;
  maxSignalingBytes: number;
  reportReasons: string[];
  serverTime: string;
};

/** The response envelope every endpoint uses. */
export type ApiEnvelope<T> = {
  success: boolean;
  message?: string;
  code?: string;
  data?: T;
};

/* -------------------------------------------------------------------------
   Signaling messages. `matched` and `partner_left` are pushed by the server;
   offer/answer/ice are relayed between the two participants.
   ------------------------------------------------------------------------- */

export type ServerEventType =
  | "hello"
  | "matched"
  | "partner_ready"
  | "partner_left"
  | "peer_unreachable"
  | "offer"
  | "answer"
  | "ice"
  | "pong"
  | "error";

export type ServerEvent = {
  type: ServerEventType | string;
  data?: unknown;
  serverTime: string;
};

export type HelloEvent = {
  sessionId: string;
  username: string;
  skipLockMs: number;
  maxMessageBytes: number;
  state: StatePayload;
};

export type MatchedEvent = {
  matchId: string;
  partnerUsername: string;
  initiator: boolean;
  matchedAt: string;
  skipLockMs: number;
  serverTime: string;
};

export type PartnerLeftEvent = {
  matchId: string;
  reason: EndReason;
  serverTime: string;
};

export type PeerUnreachableEvent = {
  matchId: string;
  graceMs: number;
};

export type RelayedEvent = {
  matchId: string;
  payload: unknown;
};

/** Human readable copy for the report reasons the API accepts. */
export const REPORT_REASONS: { value: string; label: string }[] = [
  { value: "nudity_or_sexual_content", label: "Nudity or sexual content" },
  { value: "harassment_or_hate", label: "Harassment, hate, or threats" },
  { value: "minor", label: "Someone who appears to be under 18" },
  { value: "spam_or_scam", label: "Spam, scam, or advertising" },
  { value: "self_harm", label: "Self-harm or crisis" },
  { value: "other", label: "Something else" },
];

/** The copy shown when a call ends, phrased for the visitor who stayed. */
export const END_REASON_COPY: Record<EndReason, { title: string; body: string }> = {
  skipped: {
    title: "Your partner moved on",
    body: "They ended the call. You can find someone new whenever you are ready.",
  },
  stopped: {
    title: "Your partner ended the call",
    body: "Nothing else was shared. Start a new match when you want to.",
  },
  left: {
    title: "Your partner left",
    body: "They closed the page. Start a new match when you want to.",
  },
  disconnected: {
    title: "The connection dropped",
    body: "One side lost its connection, so the call ended.",
  },
  session_ended: {
    title: "The call ended",
    body: "The session expired, so the call was closed.",
  },
  too_long: {
    title: "Time's up on that call",
    body: "Calls are capped so a forgotten tab cannot hold a connection open. Start a new one.",
  },
};
