import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";

import { api, apiErrorCode, apiErrorMessage, isApiError, skipLockedRemainingMs } from "./api";
import { signalingUrl } from "./config";
import { useMedia } from "./media";
import { SignalingClient } from "./signaling";
import { computeRemainingMs, describeSkipLock, nextTickDelayMs, serverOffsetMs } from "./skip";
import type {
  CallPhase,
  EndReason,
  HelloEvent,
  IceServer,
  MatchedEvent,
  PartnerLeftEvent,
  PublicConfig,
  RelayedEvent,
  ServerEvent,
  StatePayload,
} from "./types";
import { PeerSession, webRtcSupport } from "./webrtc";

/**
 * The call lifecycle.
 *
 * One state machine covers the whole journey: idle → queued → connecting →
 * connected → ended. The server is authoritative at every step — the queue,
 * the match, the five second Skip window, and the reason a call ended all come
 * from the backend. This provider decides how to *show* that state, and owns
 * the browser-only parts: the media stream, the peer connection, and the
 * signaling socket.
 */
type CallContextValue = {
  phase: CallPhase;
  matchId: string | null;
  partnerUsername: string | null;
  initiator: boolean;
  matchedAt: string | null;
  skipLockMs: number;
  remainingMs: number;
  connectionState: RTCPeerConnectionState;
  remoteStream: MediaStream | null;
  endedReason: EndReason | null;
  peerUnreachable: boolean;
  signalingConnected: boolean;
  error: string | null;
  busy: boolean;
  startMatching: (username: string) => Promise<void>;
  cancelMatching: () => Promise<void>;
  skip: () => Promise<void>;
  stop: () => Promise<void>;
  findSomeoneNew: (username?: string) => Promise<void>;
  report: (reason: string, note: string) => Promise<void>;
  clearError: () => void;
  reset: () => void;
};

const CallContext = createContext<CallContextValue | null>(null);

/** How long a dropped peer connection is given before the call is closed. */
const DISCONNECT_GRACE_MS = 12_000;

const FALLBACK_ICE_SERVERS: IceServer[] = [{ urls: "stun:stun.l.google.com:19302" }];

export function CallProvider({ children }: { children: React.ReactNode }) {
  const media = useMedia();

  const [phase, setPhase] = useState<CallPhase>("idle");
  const [matchId, setMatchId] = useState<string | null>(null);
  const [partnerUsername, setPartnerUsername] = useState<string | null>(null);
  const [initiator, setInitiator] = useState(false);
  const [matchedAt, setMatchedAt] = useState<string | null>(null);
  const [skipLockMs, setSkipLockMs] = useState(5000);
  const [remainingMs, setRemainingMs] = useState(0);
  const [connectionState, setConnectionState] = useState<RTCPeerConnectionState>("new");
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);
  const [endedReason, setEndedReason] = useState<EndReason | null>(null);
  const [peerUnreachable, setPeerUnreachable] = useState(false);
  const [signalingConnected, setSignalingConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const signalingRef = useRef<SignalingClient | null>(null);
  const peerRef = useRef<PeerSession | null>(null);
  const matchRef = useRef<string | null>(null);
  const reportableMatchRef = useRef<string | null>(null);
  const offsetRef = useRef(0);
  const iceRef = useRef<IceServer[] | null>(null);
  const lastUsernameRef = useRef<string>("");
  const disconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // True once this page has asked to be matched. It is what separates a real
  // page reload (the socket's greeting reports a call we did not start here)
  // from a first connection that happens to arrive after our own join.
  const hasJoinedThisPageRef = useRef(false);

  // Values the socket handlers need without re-creating the socket on every
  // render. Refs are kept in step by the effect below.
  const live = useRef<{ stream: MediaStream | null }>({ stream: null });

  useEffect(() => {
    live.current.stream = media.stream;
  }, [media.stream]);

  /** Close the peer connection without touching the visitor's own camera. */
  const teardownPeer = useCallback(() => {
    if (disconnectTimerRef.current !== null) {
      clearTimeout(disconnectTimerRef.current);
      disconnectTimerRef.current = null;
    }
    peerRef.current?.close();
    peerRef.current = null;
    setRemoteStream(null);
    setConnectionState("new");
    setPeerUnreachable(false);
  }, []);

  /** Move to the ended state: no call, no peer connection, reason recorded. */
  const endCall = useCallback(
    (reason: EndReason | null, message: string | null = null) => {
      teardownPeer();
      matchRef.current = null;
      setMatchId(null);
      setRemainingMs(0);
      setEndedReason(reason);
      setPhase("ended");
      if (message) {
        setError(message);
      }
    },
    [teardownPeer],
  );

  /** Build the peer connection for a fresh match and start negotiating. */
  const startPeer = useCallback(
    async (id: string, isInitiator: boolean, stream: MediaStream, iceServers: IceServer[]) => {
      peerRef.current?.close();

      const session = new PeerSession(iceServers, stream, {
        onRemoteStream: setRemoteStream,
        onIceCandidate: (candidate) => {
          signalingRef.current?.send("ice", id, candidate);
        },
        onConnectionState: (state) => {
          setConnectionState(state);

          if (state === "connected") {
            if (disconnectTimerRef.current !== null) {
              clearTimeout(disconnectTimerRef.current);
              disconnectTimerRef.current = null;
            }
            setPhase("connected");
            setPeerUnreachable(false);
            // Telling the server we are up is what moves the state from
            // "connecting" to "matched" for both participants.
            signalingRef.current?.send("ready", id);
            return;
          }

          if (state === "disconnected") {
            // ICE often recovers on its own; give it a moment before ending.
            setPeerUnreachable(true);
            if (disconnectTimerRef.current === null) {
              disconnectTimerRef.current = setTimeout(() => {
                disconnectTimerRef.current = null;
                endCall("disconnected", "The video connection dropped. Find someone new.");
              }, DISCONNECT_GRACE_MS);
            }
            return;
          }

          if (state === "failed") {
            endCall("disconnected", "The connection could not be established. Try someone new.");
          }
        },
      });

      peerRef.current = session;

      if (isInitiator) {
        try {
          const offer = await session.createOffer();
          if (!signalingRef.current?.send("offer", id, offer)) {
            endCall(null, "The signaling connection dropped before the call could start.");
          }
        } catch {
          endCall(null, "Could not start the video call. Try someone new.");
        }
      }
    },
    [endCall],
  );

  /** The username to use for a match request: the last one the visitor chose. */
  const currentUsername = useCallback(() => lastUsernameRef.current, []);


  /** Everything the server pushes over the signaling socket. */
  const handleServerEvent = useCallback(
    (event: ServerEvent) => {
      switch (event.type) {
        case "hello": {
          const data = event.data as HelloEvent | undefined;
          offsetRef.current = serverOffsetMs(event.serverTime, Date.now());
          if (typeof data?.skipLockMs === "number") {
            setSkipLockMs(data.skipLockMs);
          }

          const resumed = data?.state;
          if (
            resumed?.matchId &&
            (resumed.state === "matched" || resumed.state === "connecting") &&
            !hasJoinedThisPageRef.current
          ) {
            // A reload landed on a call this page never joined. A peer
            // connection cannot be restored from a cold start, so say goodbye
            // rather than leaving the other side looking at a frozen picture.
            signalingRef.current?.send("bye", resumed.matchId);
            endCall("left", "That call ended when the page reloaded.");
          }
          break;
        }

        case "matched": {
          const data = event.data as MatchedEvent;
          offsetRef.current = serverOffsetMs(data.serverTime ?? event.serverTime, Date.now());

          matchRef.current = data.matchId;
          reportableMatchRef.current = data.matchId;
          setMatchId(data.matchId);
          setPartnerUsername(data.partnerUsername);
          setInitiator(data.initiator);
          setMatchedAt(data.matchedAt);
          setSkipLockMs(data.skipLockMs);
          setRemainingMs(computeRemainingMs(data.matchedAt, data.skipLockMs, Date.now(), offsetRef.current));
          setEndedReason(null);
          setError(null);
          setPeerUnreachable(false);
          setPhase("connecting");

          const stream = live.current.stream;
          const iceServers = iceRef.current ?? FALLBACK_ICE_SERVERS;

          if (stream) {
            void startPeer(data.matchId, data.initiator, stream, iceServers);
          } else {
            endCall(null, "Camera access was lost, so the call could not start.");
          }
          break;
        }

        case "offer": {
          const data = event.data as RelayedEvent;
          const peer = peerRef.current;
          if (!peer || !data?.matchId) {
            break;
          }
          void peer
            .acceptOffer(data.payload as RTCSessionDescriptionInit)
            .then((answer) => {
              signalingRef.current?.send("answer", data.matchId, answer);
            })
            .catch(() => {
              endCall(null, "Could not answer the call. Try someone new.");
            });
          break;
        }

        case "answer": {
          const data = event.data as RelayedEvent;
          void peerRef.current?.acceptAnswer(data.payload as RTCSessionDescriptionInit);
          break;
        }

        case "ice": {
          const data = event.data as RelayedEvent;
          void peerRef.current?.addIceCandidate(data.payload as RTCIceCandidateInit);
          break;
        }

        case "partner_left": {
          const data = event.data as PartnerLeftEvent;
          endCall(data?.reason ?? null);
          break;
        }

        case "peer_unreachable": {
          setPeerUnreachable(true);
          break;
        }

        case "error": {
          const data = event.data as { code?: string; message?: string } | undefined;
          if (data?.code === "not_in_match") {
            endCall(null, "That call is no longer active.");
          } else if (data?.code === "rate_limited") {
            setError("Too many signaling messages. The call may take longer to connect.");
          }
          break;
        }

        default:
          break;
      }
    },
    [endCall, startPeer],
  );

  // The socket is created once, so its handlers read the latest callback
  // through a ref instead of being rebuilt (which would drop the connection).
  const handleServerEventRef = useRef(handleServerEvent);
  useEffect(() => {
    handleServerEventRef.current = handleServerEvent;
  }, [handleServerEvent]);

  /** Open the signaling socket if it is not already open. */
  const ensureSignaling = useCallback((): SignalingClient => {
    if (signalingRef.current) {
      if (!signalingRef.current.isOpen()) {
        signalingRef.current.connect();
      }
      return signalingRef.current;
    }

    const client = new SignalingClient(signalingUrl(), {
      onEvent: (event) => handleServerEventRef.current(event),
      onOpen: () => setSignalingConnected(true),
      onClose: (willRetry) => {
        setSignalingConnected(false);
        if (!willRetry) {
          setError("Lost the connection to Strangerly. Reload the page to continue.");
        }
      },
    });

    signalingRef.current = client;
    client.connect();
    return client;
  }, []);

  /** Fetch the ICE servers once; fall back to a public STUN server. */
  const ensureIceServers = useCallback(async (): Promise<IceServer[]> => {
    if (iceRef.current) {
      return iceRef.current;
    }

    let servers: IceServer[] = FALLBACK_ICE_SERVERS;

    try {
      const config: PublicConfig = await api.getConfig();
      if (config.iceServers?.length) {
        servers = config.iceServers;
      }
      if (typeof config.skipLockSeconds === "number" && config.skipLockSeconds > 0) {
        setSkipLockMs(Math.round(config.skipLockSeconds * 1000));
      }
    } catch {
      // A public STUN server still works on most home networks; the honest
      // limitation about restrictive NATs is documented in the README.
      servers = FALLBACK_ICE_SERVERS;
    }

    iceRef.current = servers;
    return servers;
  }, []);


  /**
   * Start matching. Media is requested first, from this deliberate action, so
   * the visitor is never queued before their camera is actually on.
   */
  const startMatching = useCallback(
    async (username: string) => {
      setError(null);
      setEndedReason(null);
      setBusy(true);
      lastUsernameRef.current = username;
      hasJoinedThisPageRef.current = true;

      try {
        const support = webRtcSupport();
        if (!support.ok) {
          setError(support.message ?? "This browser cannot make video calls.");
          setPhase("idle");
          return;
        }

        const stream = media.stream ?? (await media.request());
        if (!stream) {
          setError(media.message ?? "Camera and microphone access is needed before matching.");
          setPhase("idle");
          return;
        }
        live.current.stream = stream;

        const iceServers = await ensureIceServers();
        ensureSignaling();

        let payload: StatePayload;
        try {
          payload = await api.join(username);
        } catch (cause) {
          const code = apiErrorCode(cause);

          if (code === "already_queued") {
            // Another tab of this visit is already waiting: show that queue.
            payload =
              (isApiError(cause) ? (cause.data as StatePayload | undefined) : undefined) ?? (await api.status());
          } else if (code === "already_in_match") {
            // A match survived a reload. End it and ask once more.
            const stale = isApiError(cause) ? (cause.data as StatePayload | undefined) : undefined;
            if (stale?.matchId) {
              signalingRef.current?.send("bye", stale.matchId);
            }
            payload = await api.join(username);
          } else {
            throw cause;
          }
        }

        if (payload.matchId) {
          // Matched straight away: there was already somebody waiting.
          matchRef.current = payload.matchId;
          reportableMatchRef.current = payload.matchId;
          setMatchId(payload.matchId);
          setPartnerUsername(payload.partnerUsername ?? null);
          setInitiator(payload.initiator);
          setMatchedAt(payload.matchedAt ?? null);
          setSkipLockMs(payload.skipLockMs);
          setRemainingMs(
            computeRemainingMs(payload.matchedAt, payload.skipLockMs, Date.now(), offsetRef.current),
          );
          setPhase("connecting");
          await startPeer(payload.matchId, payload.initiator, stream, iceServers);
        } else {
          setPhase("queued");
        }
      } catch (cause) {
        setError(apiErrorMessage(cause));
        setPhase("idle");
      } finally {
        setBusy(false);
      }
    },
    [ensureIceServers, ensureSignaling, media, startPeer],
  );

  /** Leave the queue. Only meaningful while waiting. */
  const cancelMatching = useCallback(async () => {
    setBusy(true);
    try {
      await api.cancel();
    } catch {
      // Whatever the server said, the visitor asked to stop waiting.
    } finally {
      setPhase("idle");
      setBusy(false);
    }
  }, []);

  /**
   * Skip. The server refuses this inside the five second window and replies
   * with the authoritative remaining time, which is then shown instead of the
   * local estimate.
   */
  const skip = useCallback(async () => {
    const id = matchRef.current;
    if (!id) {
      return;
    }

    setBusy(true);
    try {
      await api.skip(id);
      endCall("skipped");
    } catch (cause) {
      const remaining = skipLockedRemainingMs(cause);

      if (typeof remaining === "number") {
        setRemainingMs(remaining);
        setError(remaining > 0 ? describeSkipLock(remaining) : null);
      } else if (apiErrorCode(cause) === "match_not_found") {
        endCall(null, "That call already ended.");
      } else {
        setError(apiErrorMessage(cause));
      }
    } finally {
      setBusy(false);
    }
  }, [endCall]);

  /**
   * Stop. Always available: it works during the Skip window, during
   * connecting, and when the call already ended.
   */
  const stop = useCallback(async () => {
    const id = matchRef.current;
    setBusy(true);

    try {
      if (id) {
        await api.stop(id);
      }
    } catch {
      // The call is over from the visitor's point of view either way; the
      // server will clean up its own copy through the janitor.
    } finally {
      endCall("stopped");
      setBusy(false);
    }
  }, [endCall]);

  /** Start a fresh match from the ended state. */
  const findSomeoneNew = useCallback(
    async (username?: string) => {
      const name = username ?? lastUsernameRef.current;
      if (!name) {
        setError("Choose a name before you start matching.");
        setPhase("idle");
        return;
      }
      await startMatching(name);
    },
    [startMatching],
  );

  /** Report the current or just-ended match. Never blocked by the Skip lock. */
  const report = useCallback(async (reason: string, note: string) => {
    const id = matchRef.current ?? reportableMatchRef.current;
    if (!id) {
      throw new Error("There is no call to report.");
    }
    await api.report(id, reason, note);
  }, []);

  const clearError = useCallback(() => setError(null), []);

  const reset = useCallback(() => {
    teardownPeer();
    matchRef.current = null;
    reportableMatchRef.current = null;
    setMatchId(null);
    setPartnerUsername(null);
    setEndedReason(null);
    setError(null);
    setRemainingMs(0);
    setPhase("idle");
  }, [teardownPeer]);


  /* ---------------------------------------------------------------------
     Effects: the countdown, the goodbye on unload, track-ended handling, and
     the final cleanup.
     --------------------------------------------------------------------- */

  // The Skip countdown is presentation only. It is recomputed from the
  // server's matchedAt on every tick, and it is corrected by any 409 the
  // server sends back.
  useEffect(() => {
    if (phase !== "connecting" && phase !== "connected") {
      return;
    }
    if (!matchedAt) {
      return;
    }

    let timer: ReturnType<typeof setTimeout>;

    const tick = () => {
      const next = computeRemainingMs(matchedAt, skipLockMs, Date.now(), offsetRef.current);
      setRemainingMs(next);
      timer = setTimeout(tick, nextTickDelayMs(next));
    };

    tick();
    return () => clearTimeout(timer);
  }, [phase, matchedAt, skipLockMs]);

  // Say goodbye when the tab is hidden or closed, so the partner is not left
  // looking at a frozen picture while the janitor waits for a timeout.
  useEffect(() => {
    const handlePageHide = () => {
      const id = matchRef.current;
      if (id) {
        signalingRef.current?.send("bye", id);
      }
    };

    window.addEventListener("pagehide", handlePageHide);
    return () => window.removeEventListener("pagehide", handlePageHide);
  }, []);

  // A camera that is switched off from the browser's own UI (or unplugged)
  // ends its track without telling us. Explain it instead of showing a black
  // rectangle with no context.
  useEffect(() => {
    const stream = media.stream;
    if (!stream) {
      return;
    }

    const handleEnded = () => {
      setError("Your camera or microphone stopped. Turn it back on, or stop the call.");
    };

    const tracks = [...stream.getVideoTracks(), ...stream.getAudioTracks()];
    for (const track of tracks) {
      track.addEventListener("ended", handleEnded);
    }

    return () => {
      for (const track of tracks) {
        track.removeEventListener("ended", handleEnded);
      }
    };
  }, [media.stream]);

  // Close the peer connection and the socket when the app itself unmounts.
  useEffect(() => {
    return () => {
      peerRef.current?.close();
      peerRef.current = null;
      signalingRef.current?.close();
      signalingRef.current = null;
    };
  }, []);

  const value = useMemo<CallContextValue>(
    () => ({
      phase,
      matchId,
      partnerUsername,
      initiator,
      matchedAt,
      skipLockMs,
      remainingMs,
      connectionState,
      remoteStream,
      endedReason,
      peerUnreachable,
      signalingConnected,
      error,
      busy,
      startMatching,
      cancelMatching,
      skip,
      stop,
      findSomeoneNew,
      report,
      clearError,
      reset,
    }),
    [
      phase,
      matchId,
      partnerUsername,
      initiator,
      matchedAt,
      skipLockMs,
      remainingMs,
      connectionState,
      remoteStream,
      endedReason,
      peerUnreachable,
      signalingConnected,
      error,
      busy,
      startMatching,
      cancelMatching,
      skip,
      stop,
      findSomeoneNew,
      report,
      clearError,
      reset,
    ],
  );

  return <CallContext.Provider value={value}>{children}</CallContext.Provider>;
}

export function useCall(): CallContextValue {
  const context = useContext(CallContext);
  if (!context) {
    throw new Error("useCall must be used inside a CallProvider");
  }
  return context;
}

/** True while the visitor is in a live call (connecting or connected). */
export function useInLiveCall(): boolean {
  const { phase } = useCall();
  return phase === "connecting" || phase === "connected";
}

