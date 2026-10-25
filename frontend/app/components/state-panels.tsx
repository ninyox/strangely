import { AlertTriangle, Flag, PhoneOff, RefreshCw, Search, UserRound } from "lucide-react";

import { END_REASON_COPY, type EndReason } from "../../lib/types";
import { cn } from "../../lib/utils";
import { LiveDot, SignalRings, Spinner } from "./spinner";
import { Button } from "./ui/button";

/** The dark surface every call state is built on. */
function Panel({
  children,
  className,
  live = false,
}: {
  children: React.ReactNode;
  className?: string;
  live?: boolean;
}) {
  return (
    <div
      className={cn(
        "grain relative overflow-hidden rounded-stage border border-stage-line bg-stage-raised p-7 text-stage-text shadow-stage sm:p-10",
        className,
      )}
    >
      <div className="grain-overlay" aria-hidden="true" />
      {live ? (
        <span className="relative mb-6 inline-flex items-center gap-2">
          <LiveDot />
          <span className="mono-label text-stage-muted">Searching</span>
        </span>
      ) : null}
      <div className="relative">{children}</div>
    </div>
  );
}

/** Waiting in the queue: calm, animated, with a way out. */
export function SearchingPanel({
  queuePosition,
  onCancel,
  busy,
}: {
  queuePosition: number;
  onCancel: () => void;
  busy: boolean;
}) {
  return (
    <Panel live aria-live="polite">
      <div className="flex flex-col items-center gap-8 py-4 text-center">
        <div className="relative grid size-40 place-items-center">
          <SignalRings className="size-40" tone="stage" />
          <span className="relative grid size-14 place-items-center rounded-full border border-stage-line bg-stage">
            <Search className="size-5 text-signal-bright" aria-hidden="true" />
          </span>
        </div>

        <div className="flex flex-col gap-2">
          <h2 className="font-display text-display-md text-stage-text">Looking for someone</h2>
          <p className="measure mx-auto text-body text-stage-muted">
            You are in the queue. The next person to arrive is matched with you — the queue is first
            in, first out.
          </p>
          {queuePosition > 0 ? (
            <p className="mono-label text-stage-muted">Position {queuePosition}</p>
          ) : null}
        </div>

        <Button type="button" variant="stage" onClick={onCancel} disabled={busy}>
          Cancel and go back
        </Button>
      </div>
    </Panel>
  );
}

/** The moment between "matched" and a working video connection. */
export function PreparingPanel({ message, step }: { message?: string; step?: string }) {
  return (
    <Panel aria-live="polite">
      <div className="flex flex-col items-center gap-6 py-8 text-center">
        <Spinner tone="stage" />
        <div className="flex flex-col gap-2">
          <h2 className="font-display text-display-sm text-stage-text">
            {message ?? "Getting the call ready"}
          </h2>
          <p className="mono-label text-stage-muted">{step ?? "Opening the connection"}</p>
        </div>
      </div>
    </Panel>
  );
}

/** A call that could not start: say why, offer both ways forward. */
export function FailedPanel({
  message,
  onRetry,
  onLeave,
  busy,
}: {
  message: string;
  onRetry: () => void;
  onLeave: () => void;
  busy: boolean;
}) {
  return (
    <Panel>
      <div className="flex flex-col gap-5">
        <span className="mono-label flex items-center gap-2 text-signal-bright">
          <AlertTriangle className="size-4" aria-hidden="true" />
          Call not started
        </span>
        <h2 className="font-display text-display-md text-stage-text">We could not start the call</h2>
        <p className="measure text-body text-stage-muted">{message}</p>
        <p className="measure text-small text-stage-muted">
          Nothing is running and you are not in the queue.
        </p>

        <div className="mt-1 flex flex-wrap gap-3">
          <Button type="button" onClick={onRetry} disabled={busy}>
            <RefreshCw aria-hidden="true" />
            Try again
          </Button>
          <Button type="button" variant="stage" onClick={onLeave}>
            Back to start
          </Button>
        </div>
      </div>
    </Panel>
  );
}

/** A call that finished, with the reason the server reported. */
export function EndedPanel({
  reason,
  message,
  onFindSomeoneNew,
  onLeave,
  onReport,
  busy,
}: {
  reason: EndReason | null;
  message: string | null;
  onFindSomeoneNew: () => void;
  onLeave: () => void;
  onReport: () => void;
  busy: boolean;
}) {
  const copy = reason
    ? END_REASON_COPY[reason]
    : { title: "Call ended", body: "That conversation is over. You can start a new one." };

  return (
    <Panel>
      <div className="flex flex-col gap-5">
        <span className="mono-label text-stage-muted">Call ended</span>
        <h2 className="font-display text-display-md text-stage-text">{copy.title}</h2>
        <p className="measure text-body text-stage-muted">{copy.body}</p>

        {message ? (
          <p className="measure rounded-control border border-stage-line bg-white/5 px-4 py-3 text-small text-stage-text">
            {message}
          </p>
        ) : null}

        <div className="mt-1 flex flex-wrap gap-3">
          <Button type="button" onClick={onFindSomeoneNew} disabled={busy}>
            <UserRound aria-hidden="true" />
            Find someone new
          </Button>
          <Button type="button" variant="stage" onClick={onLeave} disabled={busy}>
            <PhoneOff aria-hidden="true" />
            Leave
          </Button>
          <Button
            type="button"
            variant="ghost"
            onClick={onReport}
            className="text-stage-muted hover:bg-white/8 hover:text-stage-text"
          >
            <Flag aria-hidden="true" />
            Report this call
          </Button>
        </div>

        <p className="mt-2 border-t border-stage-line pt-5 text-caption text-stage-muted">
          Nothing from that call was recorded or stored. You can still report it for a short while.
        </p>
      </div>
    </Panel>
  );
}

/** Something needs saying mid-call but the call itself is fine. */
export function CallWarning({ message }: { message: string }) {
  return (
    <p
      role="alert"
      className="flex items-start gap-2.5 rounded-control border border-stage-line border-l-2 border-l-warning bg-white/5 px-4 py-3 text-small text-stage-text"
    >
      <AlertTriangle className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden="true" />
      <span>{message}</span>
    </p>
  );
}

