import { useEffect, useRef } from "react";

import { cn } from "../../lib/utils";

/**
 * A video surface.
 *
 * Both the remote picture and the local preview go through here so the two can
 * never drift. The stream is attached with an effect (React never sets a
 * MediaStream through a prop) and an empty tile shows a calm placeholder
 * instead of a broken element.
 */
export function VideoTile({
  stream,
  muted = false,
  mirrored = false,
  label,
  placeholder,
  className,
  videoClassName,
}: {
  stream: MediaStream | null;
  muted?: boolean;
  mirrored?: boolean;
  label?: string;
  placeholder?: React.ReactNode;
  className?: string;
  videoClassName?: string;
}) {
  const videoRef = useRef<HTMLVideoElement | null>(null);

  useEffect(() => {
    const element = videoRef.current;
    if (!element) {
      return;
    }

    if (element.srcObject !== stream) {
      element.srcObject = stream;
    }

    if (stream) {
      const played = element.play();
      if (played && typeof played.catch === "function") {
        played.catch(() => undefined);
      }
    }
  }, [stream]);

  const hasVideo = Boolean(stream && stream.getVideoTracks().length > 0);

  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-stage bg-stage",
        // A very soft vignette so the picture sits in the surface rather than
        // looking pasted onto it.
        "before:pointer-events-none before:absolute before:inset-0 before:z-10 before:bg-[radial-gradient(120%_90%_at_50%_-10%,rgba(255,255,255,0.07),transparent_58%)]",
        className,
      )}
    >
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted={muted}
        aria-label={label}
        className={cn(
          "size-full object-cover transition-opacity duration-500 ease-settle",
          hasVideo ? "opacity-100" : "opacity-0",
          mirrored && "-scale-x-100",
          videoClassName,
        )}
      />

      {!hasVideo ? (
        <div className="absolute inset-0 grid place-items-center p-6 text-center text-stage-text">
          {placeholder}
        </div>
      ) : null}
    </div>
  );
}
