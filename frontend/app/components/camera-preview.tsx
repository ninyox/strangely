import { useEffect, useRef, useState } from "react";
import { Camera, CameraOff, Mic, MicOff } from "lucide-react";

import { useMedia } from "../../lib/media";
import { cn } from "../../lib/utils";
import { Notice } from "./notice";
import { LiveDot, Spinner } from "./spinner";
import { Button } from "./ui/button";
import { VideoTile } from "./video-tile";

/** Elapsed time since the camera went live, as a broadcast-style mm:ss. */
function useTimecode(running: boolean): string {
  const [seconds, setSeconds] = useState(0);
  const startedAt = useRef<number | null>(null);

  useEffect(() => {
    if (!running) {
      startedAt.current = null;
      setSeconds(0);
      return;
    }

    startedAt.current = Date.now();
    const timer = window.setInterval(() => {
      if (startedAt.current !== null) {
        setSeconds(Math.floor((Date.now() - startedAt.current) / 1000));
      }
    }, 1000);

    return () => window.clearInterval(timer);
  }, [running]);

  const minutes = String(Math.floor(seconds / 60)).padStart(2, "0");
  const rest = String(seconds % 60).padStart(2, "0");
  return `${minutes}:${rest}`;
}

/**
 * The camera panel.
 *
 * Styled as a broadcast monitor — a labelled header with a live indicator and a
 * running timecode, then the picture, then the controls. It is the product's
 * only imagery, and it is real: what you see here is what the other person will
 * see, from the same component the call screen uses.
 */
export function CameraPreview() {
  const media = useMedia();
  const live = media.status === "ready";
  const timecode = useTimecode(live);

  const tone =
    media.status === "denied" || media.status === "busy" || media.status === "error"
      ? "error"
      : media.status === "unavailable" || media.status === "insecure"
        ? "warning"
        : "info";

  return (
    <div className="flex flex-col gap-3">
      <div className="overflow-hidden rounded-control border border-hairline-strong bg-surface">
        {/* Monitor chrome */}
        <div className="flex items-center justify-between gap-3 border-b border-hairline bg-clay/60 px-3 py-2">
          <span className="mono-label flex items-center gap-2 text-ink-muted">
            {live ? (
              <>
                <LiveDot />
                Preview live
              </>
            ) : (
              <>
                <span aria-hidden="true" className="size-1.5 rounded-full bg-hairline-strong" />
                Camera off
              </>
            )}
          </span>
          <span className={cn("mono-label tabular text-ink-muted", !live && "opacity-40")}>
            {timecode}
          </span>
        </div>

        <VideoTile
          stream={media.stream}
          muted
          mirrored
          label="Your camera preview"
          className="aspect-video w-full rounded-none"
          placeholder={
            <div className="flex flex-col items-center gap-3 text-center">
              {media.status === "requesting" ? (
                <Spinner tone="stage" label="Waiting for permission…" />
              ) : (
                <>
                  <span className="grid size-12 place-items-center rounded-full border border-white/20 bg-white/8">
                    <CameraOff className="size-5 text-stage-text" aria-hidden="true" />
                  </span>
                  <span className="text-small text-stage-muted">
                    Your camera stays off until you turn it on
                  </span>
                </>
              )}
            </div>
          }
        />

        {/* Controls */}
        <div className="flex flex-wrap items-center gap-2 border-t border-hairline bg-clay/60 px-3 py-2.5">
          {live ? (
            <>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={media.toggleVideo}
                aria-pressed={!media.videoEnabled}
              >
                {media.videoEnabled ? <Camera aria-hidden="true" /> : <CameraOff aria-hidden="true" />}
                {media.videoEnabled ? "Camera on" : "Camera off"}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={media.toggleAudio}
                aria-pressed={!media.audioEnabled}
              >
                {media.audioEnabled ? <Mic aria-hidden="true" /> : <MicOff aria-hidden="true" />}
                {media.audioEnabled ? "Mic on" : "Muted"}
              </Button>
              <Button type="button" variant="ghost" size="sm" className="ml-auto" onClick={media.stop}>
                Turn off
              </Button>
            </>
          ) : (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void media.request()}
              disabled={media.status === "requesting"}
            >
              {media.status === "requesting" ? <Spinner /> : <Camera aria-hidden="true" />}
              Turn on camera &amp; mic
            </Button>
          )}
        </div>
      </div>

      {media.message ? (
        <Notice tone={tone} title="Camera and microphone">
          <p>{media.message}</p>
          {media.status === "denied" ? (
            <p className="mt-2">
              Look for the camera icon in your browser&apos;s address bar, allow access, then try
              again.
            </p>
          ) : null}
        </Notice>
      ) : null}
    </div>
  );
}
