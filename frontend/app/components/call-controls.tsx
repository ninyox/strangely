import { Camera, CameraOff, Flag, Mic, MicOff, PhoneOff, SkipForward } from "lucide-react";

import { useMedia } from "../../lib/media";
import { describeSkipLock, isSkipLocked } from "../../lib/skip";
import { cn } from "../../lib/utils";
import { CountdownRing } from "./countdown-ring";
import { Button } from "./ui/button";

/**
 * The call controls.
 *
 * Five actions in one bar, always in the same order. Mute, Camera and Report
 * collapse to icons on a phone (with aria-labels) so Skip and Stop — the two
 * that decide how a call ends — keep their words. Skip carries the ring
 * countdown; Stop and Report never dim.
 */
export function CallControls({
  remainingMs,
  skipLockMs,
  onSkip,
  onStop,
  onReport,
  busy = false,
}: {
  remainingMs: number;
  skipLockMs: number;
  onSkip: () => void;
  onStop: () => void;
  onReport: () => void;
  busy?: boolean;
}) {
  const media = useMedia();
  const skipLocked = isSkipLocked(remainingMs);
  const progress = skipLockMs > 0 ? Math.min(1, Math.max(0, remainingMs / skipLockMs)) : 0;
  const seconds = Math.max(1, Math.ceil(remainingMs / 1000));

  return (
    <div className="sticky bottom-0 z-30 -mx-4 mt-1 border-t border-stage-line bg-stage/95 px-4 pb-[max(0.75rem,env(safe-area-inset-bottom))] pt-3 backdrop-blur-md sm:mx-0 sm:rounded-stage sm:border sm:px-4 sm:pb-4">
      <div className="flex items-center justify-center gap-1.5 sm:gap-2.5">
        <Button
          type="button"
          variant="stage"
          onClick={media.toggleAudio}
          aria-pressed={!media.audioEnabled}
          aria-label={media.audioEnabled ? "Mute your microphone" : "Unmute your microphone"}
          disabled={!media.stream}
          className="px-3 sm:px-4"
        >
          {media.audioEnabled ? <Mic aria-hidden="true" /> : <MicOff aria-hidden="true" />}
          <span className="hidden sm:inline">{media.audioEnabled ? "Mute" : "Unmute"}</span>
        </Button>

        <Button
          type="button"
          variant="stage"
          onClick={media.toggleVideo}
          aria-pressed={!media.videoEnabled}
          aria-label={media.videoEnabled ? "Turn your camera off" : "Turn your camera on"}
          disabled={!media.stream}
          className="px-3 sm:px-4"
        >
          {media.videoEnabled ? <Camera aria-hidden="true" /> : <CameraOff aria-hidden="true" />}
          <span className="hidden sm:inline">{media.videoEnabled ? "Camera" : "Camera off"}</span>
        </Button>

        <span aria-hidden="true" className="mx-0.5 h-8 w-px bg-stage-line sm:mx-1" />

        <Button
          type="button"
          variant={skipLocked ? "stage" : "primary"}
          onClick={onSkip}
          // The button stays operable while locked: pressing it is answered by
          // the server with the remaining time, and that answer is announced.
          // The description below carries the rule for assistive technology.
          aria-describedby="skip-lock-hint"
          title={skipLocked ? describeSkipLock(remainingMs) : "End this call and find someone new"}
          className={cn(
            "px-3 sm:px-5",
            skipLocked && "cursor-not-allowed border-stage-line bg-white/6 text-stage-muted hover:border-stage-line hover:bg-white/6",
          )}
        >
          {skipLocked ? (
            <CountdownRing progress={progress} size={18} tone="stage" />
          ) : (
            <SkipForward aria-hidden="true" />
          )}
          <span className="tabular">{skipLocked ? `Skip ${seconds}…` : "Skip"}</span>
        </Button>

        <Button type="button" variant="danger" onClick={onStop} disabled={busy} className="px-3 sm:px-5">
          <PhoneOff aria-hidden="true" />
          Stop
        </Button>

        <Button
          type="button"
          variant="stage"
          onClick={onReport}
          aria-label="Report this call"
          className="px-3 sm:px-4"
        >
          <Flag aria-hidden="true" />
          <span className="hidden sm:inline">Report</span>
        </Button>
      </div>

      <p id="skip-lock-hint" className="mt-2.5 text-center text-caption text-stage-muted" aria-live="polite">
        {skipLocked
          ? `${describeSkipLock(remainingMs)} Stop, report, mute and camera all work right now.`
          : "Skip ends the call for both of you and puts you back in the queue."}
      </p>
    </div>
  );
}
