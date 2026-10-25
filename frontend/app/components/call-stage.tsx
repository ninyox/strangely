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
 */
export function CallStage({
  remoteStream,
  localStream,
  partnerUsername,
  phase,
  connectionState,
  signalingConnected,
  peerUnreachable,
}: {
  remoteStream: MediaStream | null;
  localStream: MediaStream | null;
  partnerUsername: string | null;
  phase: "connecting" | "connected";
  connectionState: RTCPeerConnectionState;
  signalingConnected: boolean;
  peerUnreachable: boolean;
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
    <div className="relative overflow-hidden rounded-stage border border-stage-line bg-stage shadow-stage">
      <div className="relative">
        <VideoTile
          stream={remoteStream}
          label={partnerUsername ? `${partnerUsername}'s video` : "Your partner's video"}
          className="aspect-[4/3] w-full rounded-none sm:aspect-video"
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

        {/* Who you are talking to. */}
        <div className="absolute left-3 top-3 flex max-w-[calc(100%-1.5rem)] items-center gap-2.5 rounded-control border border-stage-line bg-stage/85 px-3 py-2 backdrop-blur-md">
          <LiveDot tone={stateLabel === "Live" ? "signal" : "success"} />
          <span className="truncate font-display text-[1.0625rem] font-medium text-stage-text">
            {partnerUsername ?? "Someone new"}
          </span>
        </div>

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
