import { Camera, Mic, PhoneOff, SkipForward } from "lucide-react";

import { CountdownRing } from "./countdown-ring";
import { LiveDot, SignalRings } from "./spinner";
import { VideoTile } from "./video-tile";

/**
 * The call stage, shown as it looks while waiting for a match.
 *
 * This is not a mock-up: it is built from the same VideoTile, rings and controls
 * the live call screen uses, rendered in its empty state. It is decorative here
 * (aria-hidden, not focusable) because the real thing lives on the call screen.
 */
export function StagePreview() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none select-none overflow-hidden rounded-stage border border-stage-line bg-stage-raised shadow-stage"
    >
      {/* Monitor chrome */}
      <div className="flex items-center justify-between gap-3 border-b border-stage-line px-4 py-2.5">
        <span className="mono-label flex items-center gap-2 text-stage-muted">
          <LiveDot />
          Stage
        </span>
        <span className="mono-label tabular text-stage-muted">00:00</span>
      </div>

      <div className="relative p-3 sm:p-4">
        <VideoTile
          stream={null}
          label="Waiting for a partner"
          className="aspect-[16/10] w-full"
          placeholder={
            <div className="relative flex flex-col items-center gap-3">
              <SignalRings className="size-40" tone="signal" />
              <p className="relative font-display text-display-sm text-stage-text">
                Waiting for someone
              </p>
              <p className="relative mono-label text-stage-muted">Queue · position 1</p>
            </div>
          }
        />

        {/* Local preview */}
        <div className="absolute bottom-6 right-6 w-24 rounded-control border border-stage-line bg-stage p-1 sm:bottom-7 sm:right-7 sm:w-32">
          <VideoTile
            stream={null}
            muted
            label="Your video"
            className="aspect-video w-full rounded-chip"
            placeholder={<p className="mono-label text-stage-muted">You</p>}
          />
        </div>
      </div>

      {/* Controls, in the same order and shape as the real ones. */}
      <div className="flex items-center justify-between gap-2 border-t border-stage-line px-3 py-3 sm:px-4">
        <div className="flex items-center gap-2">
          <span className="grid size-10 place-items-center rounded-control border border-white/15 bg-white/8 text-stage-text">
            <Mic className="size-[18px]" />
          </span>
          <span className="grid size-10 place-items-center rounded-control border border-white/15 bg-white/8 text-stage-text">
            <Camera className="size-[18px]" />
          </span>
        </div>

        <span className="flex h-10 items-center gap-2 rounded-control bg-white/10 px-3 text-small font-medium text-stage-text">
          <CountdownRing progress={0.62} size={18} tone="stage" />
          Skip in 3…
        </span>

        <span className="flex h-10 items-center gap-2 rounded-control bg-signal px-3.5 text-small font-medium text-white">
          <PhoneOff className="size-[18px]" />
          Stop
        </span>
      </div>

      <p className="flex items-center justify-center gap-2 border-t border-stage-line px-4 py-2.5 text-caption text-stage-muted">
        <SkipForward className="size-3.5" />
        Skip unlocks five seconds in, so nobody is dropped before they are seen
      </p>
    </div>
  );
}
