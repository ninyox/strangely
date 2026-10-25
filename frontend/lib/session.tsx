import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";

import { api, apiErrorMessage } from "./api";
import { STORAGE_KEYS } from "./config";
import { normalizeUsername } from "./username";

/**
 * The anonymous visit.
 *
 * There is no account, no email, and no password. The backend sets an opaque
 * HttpOnly cookie; this provider only records that a session exists and which
 * display name the visitor chose. The name is the only thing kept in
 * localStorage, and it is a display name — never a credential and never proof
 * of who anybody is.
 */
type SessionStatus = "loading" | "ready" | "error";

type SessionContextValue = {
  status: SessionStatus;
  sessionId: string | null;
  username: string;
  setUsername: (value: string) => void;
  error: string | null;
  retry: () => void;
  /** The validated name to send with a match request, or null when unusable. */
  readyUsername: () => string | null;
};

const SessionContext = createContext<SessionContextValue | null>(null);

function readStoredUsername(): string {
  try {
    return window.localStorage.getItem(STORAGE_KEYS.username) ?? "";
  } catch {
    // Private mode and some embedded webviews refuse localStorage. A name is
    // not worth breaking the page over.
    return "";
  }
}

function storeUsername(value: string) {
  try {
    if (value.trim() === "") {
      window.localStorage.removeItem(STORAGE_KEYS.username);
      return;
    }
    window.localStorage.setItem(STORAGE_KEYS.username, value);
  } catch {
    // Ignored on purpose: see readStoredUsername.
  }
}

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>("loading");
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [username, setUsernameState] = useState<string>(() => readStoredUsername());
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  // Guards against a second create call from a double effect in development.
  const started = useRef(false);

  useEffect(() => {
    let cancelled = false;
    started.current = true;
    setStatus("loading");
    setError(null);

    api
      .createSession()
      .then((session) => {
        if (cancelled) {
          return;
        }
        setSessionId(session.sessionId);
        setStatus("ready");

        // A resumed session may already carry a name the visitor chose before
        // a reload; only adopt it when this browser has nothing stored.
        setUsernameState((current) => {
          if (current.trim() !== "" || !session.username) {
            return current;
          }
          storeUsername(session.username);
          return session.username;
        });
      })
      .catch((cause) => {
        if (cancelled) {
          return;
        }
        setError(apiErrorMessage(cause, "Could not start a session. Please try again."));
        setStatus("error");
      });

    return () => {
      cancelled = true;
    };
  }, [attempt]);

  const setUsername = useCallback((value: string) => {
    setUsernameState(value);
    storeUsername(value);
  }, []);

  const readyUsername = useCallback(() => {
    const result = normalizeUsername(username);
    return result.ok ? result.value : null;
  }, [username]);

  const retry = useCallback(() => setAttempt((value) => value + 1), []);

  const value = useMemo<SessionContextValue>(
    () => ({ status, sessionId, username, setUsername, error, retry, readyUsername }),
    [status, sessionId, username, setUsername, error, retry, readyUsername],
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionContextValue {
  const context = useContext(SessionContext);
  if (!context) {
    throw new Error("useSession must be used inside a SessionProvider");
  }
  return context;
}
