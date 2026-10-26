import { Maximize2, Minimize2 } from "lucide-react";

import { cn } from "../../lib/utils";
import { LiveDot, SignalRings } from "./spinner";
import { VideoTile } from "./video-tile";

/**
 * The call itself.
 *
 * The remote participant owns the screen. Their name sits over the picture with
 * a live indicator, the local preview floats in the corner, and the strip below
 * carries the connection state — including the honest note that a display name
 * is not an identity.
 *
 * `expanded` is the full screen mode, native or filled: the frame disappears and
 * the picture takes every pixel the controls do not need. The toggle lives on
 * the picture rather than in the control bar, which keeps the five call actions
 * in one predictable order.
 */
export function CallStage({
  remoteStream,
  localStream,
  partnerUsername,
  phase,
  connectionState,
  signalingConnected,
  peerUnreachable,
  expanded = false,
  onToggleFullscreen,
}: {
  remoteStream: MediaStream | null;
  localStream: MediaStream | null;
  partnerUsername: string | null;
  phase: "connecting" | "connected";
  connectionState: RTCPeerConnectionState;
  signalingConnected: boolean;
  peerUnreachable: boolean;
  expanded?: boolean;
  onToggleFullscreen: () => void;
}) {
  const hasRemoteVideo = Boolean(remoteStream && remoteStream.getVideoTracks().length > 0);
  const stateLabel = peerUnreachable
    ? "Reconnecting"
    : phase === "connecting" || !hasRemoteVideo
      ? "Connecting"
      : connectionState === "connected"
        ? "Live"
        : connectionState;

  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-stage border border-stage-line bg-stage shadow-stage",
        // Expanded: the call owns the screen, so the frame goes and the picture
        // takes every pixel the control bar below does not need.
        expanded && "flex min-h-0 flex-1 flex-col rounded-none border-0 shadow-none",
      )}
    >
      <div className={cn("relative", expanded && "min-h-0 flex-1")}>
        <VideoTile
          stream={remoteStream}
          label={partnerUsername ? `${partnerUsername}'s video` : "Your partner's video"}
          className={cn(
            "aspect-[4/3] w-full rounded-none sm:aspect-video",
            expanded && "aspect-auto h-full",
          )}
          // Nothing is cropped when the picture owns the screen: a call is
          // watched whole, letterboxed, not filled edge to edge.
          videoClassName={expanded ? "object-contain" : undefined}
          placeholder={
            <div className="relative flex flex-col items-center gap-4">
              <SignalRings className="size-44" tone="stage" />
              <div className="relative flex flex-col items-center gap-1.5">
                <p className="font-display text-display-sm text-stage-text">
                  {peerUnreachable ? "Reconnecting…" : "Connecting video…"}
                </p>
                <p className="mono-label text-stage-muted">
                  {partnerUsername ? `Matched with ${partnerUsername}` : "Waiting for a match"}
                </p>
              </div>
            </div>
          }
        />

        {/* Who you are talking to. The width stops short of the full screen
            control in the opposite corner. */}
        <div className="absolute left-3 top-3 flex max-w-[calc(100%-5.5rem)] items-center gap-2.5 rounded-control border border-stage-line bg-stage/85 px-3 py-2 backdrop-blur-md">
          <LiveDot tone={stateLabel === "Live" ? "signal" : "success"} />
          <span className="truncate font-display text-[1.0625rem] font-medium text-stage-text">
            {partnerUsername ?? "Someone new"}
          </span>
        </div>

        {/* Full screen. On the picture, where a viewer looks for it. */}
        <button
          type="button"
          onClick={onToggleFullscreen}
          aria-pressed={expanded}
          aria-label={expanded ? "Leave full screen" : "Fill the screen"}
          title={expanded ? "Leave full screen" : "Full screen"}
          className="absolute right-3 top-3 z-20 inline-flex size-10 items-center justify-center rounded-control border border-stage-line bg-stage/85 text-stage-text backdrop-blur-md transition-[background-color,border-color] duration-200 ease-settle hover:border-white/30 hover:bg-white/16"
        >
          {expanded ? (
            <Minimize2 className="size-4" aria-hidden="true" />
          ) : (
            <Maximize2 className="size-4" aria-hidden="true" />
          )}
        </button>

        {/* Your own picture. */}
        <div className="absolute bottom-3 right-3 w-24 sm:w-36 lg:w-44">
          <VideoTile
            stream={localStream}
            muted
            mirrored
            label="Your video"
            className="aspect-video w-full rounded-control border border-stage-line"
            placeholder={<p className="mono-label text-stage-muted">Camera off</p>}
          />
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-t border-stage-line px-4 py-3">
        <span className="flex items-center gap-2">
          <span
            className={cn(
              "size-1.5 rounded-full",
              stateLabel === "Live"
                ? "bg-signal-bright"
                : stateLabel === "Reconnecting"
                  ? "bg-warning"
                  : "bg-stage-muted",
            )}
            aria-hidden="true"
          />
          <span className="mono-label text-stage-text">{stateLabel}</span>
        </span>

        <span className="flex items-center gap-4">
          {!signalingConnected ? (
            <span className="mono-label text-warning">Signaling reconnecting…</span>
          ) : null}
          <span className="text-caption text-stage-muted">
            A display name chosen for this visit — not verified
          </span>
        </span>
      </div>

      <p className="sr-only" aria-live="polite">
        {stateLabel === "Live"
          ? `Connected to ${partnerUsername ?? "your partner"}.`
          : `${stateLabel}.`}
      </p>
    </div>
  );
}
