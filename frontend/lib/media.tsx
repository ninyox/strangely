import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";

/**
 * Camera and microphone access.
 *
 * Nothing here runs until the visitor asks for it: `request()` is only called
 * from an explicit action (the preview button, or Start matching). Every
 * failure mode is turned into copy that says what actually happened, because
 * "something went wrong" is useless when a camera is blocked or busy.
 */
export type MediaStatus =
  | "idle"
  | "requesting"
  | "ready"
  | "denied"
  | "unavailable"
  | "insecure"
  | "busy"
  | "error";

type MediaContextValue = {
  status: MediaStatus;
  stream: MediaStream | null;
  audioEnabled: boolean;
  videoEnabled: boolean;
  message: string | null;
  request: () => Promise<MediaStream | null>;
  stop: () => void;
  toggleAudio: () => void;
  toggleVideo: () => void;
};

const MediaContext = createContext<MediaContextValue | null>(null);

/** Modest constraints: good enough to be seen clearly, light enough for mobile. */
const CONSTRAINTS: MediaStreamConstraints = {
  video: {
    width: { ideal: 1280 },
    height: { ideal: 720 },
    frameRate: { ideal: 30, max: 30 },
    facingMode: "user",
  },
  audio: {
    echoCancellation: true,
    noiseSuppression: true,
    autoGainControl: true,
  },
};

function describeFailure(error: unknown): { status: MediaStatus; message: string } {
  const name = error instanceof Error ? error.name : "";

  switch (name) {
    case "NotAllowedError":
    case "SecurityError":
      return {
        status: "denied",
        message:
          "Camera and microphone access is blocked. Allow it in your browser's site settings, then try again.",
      };
    case "NotFoundError":
    case "DevicesNotFoundError":
      return {
        status: "unavailable",
        message: "No camera or microphone was found on this device.",
      };
    case "NotReadableError":
    case "TrackStartError":
      return {
        status: "busy",
        message:
          "Your camera or microphone is already in use by another app. Close it and try again.",
      };
    case "OverconstrainedError":
      return {
        status: "unavailable",
        message: "This device's camera does not support the video settings Strangerly needs.",
      };
    default:
      return {
        status: "error",
        message: "Camera and microphone access failed. Please try again.",
      };
  }
}

export function MediaProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<MediaStatus>("idle");
  const [stream, setStream] = useState<MediaStream | null>(null);
  const [audioEnabled, setAudioEnabled] = useState(true);
  const [videoEnabled, setVideoEnabled] = useState(true);
  const [message, setMessage] = useState<string | null>(null);

  const streamRef = useRef<MediaStream | null>(null);

  const request = useCallback(async (): Promise<MediaStream | null> => {
    // Reuse an existing stream instead of asking for a second one: two
    // getUserMedia calls would turn the camera on twice on some devices.
    if (streamRef.current) {
      return streamRef.current;
    }

    if (!window.isSecureContext) {
      setStatus("insecure");
      setMessage("Camera access needs a secure connection (https), except on localhost.");
      return null;
    }

    if (!navigator.mediaDevices?.getUserMedia) {
      setStatus("unavailable");
      setMessage("This browser cannot share a camera or microphone.");
      return null;
    }

    setStatus("requesting");
    setMessage(null);

    try {
      const next = await navigator.mediaDevices.getUserMedia(CONSTRAINTS);

      streamRef.current = next;
      setStream(next);
      setAudioEnabled(true);
      setVideoEnabled(true);
      setStatus("ready");
      setMessage(null);
      return next;
    } catch (error) {
      const failure = describeFailure(error);
      streamRef.current = null;
      setStream(null);
      setStatus(failure.status);
      setMessage(failure.message);
      return null;
    }
  }, []);

  const stop = useCallback(() => {
    const current = streamRef.current;
    if (current) {
      for (const track of current.getTracks()) {
        track.stop();
      }
    }
    streamRef.current = null;
    setStream(null);
    setStatus("idle");
    setMessage(null);
    setAudioEnabled(true);
    setVideoEnabled(true);
  }, []);

  const toggleAudio = useCallback(() => {
    const current = streamRef.current;
    if (!current) {
      return;
    }
    const next = !audioEnabled;
    for (const track of current.getAudioTracks()) {
      track.enabled = next;
    }
    setAudioEnabled(next);
  }, [audioEnabled]);

  const toggleVideo = useCallback(() => {
    const current = streamRef.current;
    if (!current) {
      return;
    }
    const next = !videoEnabled;
    for (const track of current.getVideoTracks()) {
      track.enabled = next;
    }
    setVideoEnabled(next);
  }, [videoEnabled]);

  // Release the camera when the page goes away, so the recording light does
  // not stay on after the visitor leaves.
  useEffect(() => {
    const handlePageHide = () => stop();
    window.addEventListener("pagehide", handlePageHide);
    return () => window.removeEventListener("pagehide", handlePageHide);
  }, [stop]);

  const value = useMemo<MediaContextValue>(
    () => ({ status, stream, audioEnabled, videoEnabled, message, request, stop, toggleAudio, toggleVideo }),
    [status, stream, audioEnabled, videoEnabled, message, request, stop, toggleAudio, toggleVideo],
  );

  return <MediaContext.Provider value={value}>{children}</MediaContext.Provider>;
}

export function useMedia(): MediaContextValue {
  const context = useContext(MediaContext);
  if (!context) {
    throw new Error("useMedia must be used inside a MediaProvider");
  }
  return context;
}
