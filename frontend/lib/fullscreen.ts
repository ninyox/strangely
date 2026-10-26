import { useCallback, useEffect, useRef, useState, type RefObject } from "react";

/**
 * Filling the screen.
 *
 * The Fullscreen API is used directly — a wrapper library such as `screenfull`
 * would be a dependency for something that is a handful of lines, and it would
 * not solve the one thing that actually needs solving: Safari on iPhone exposes
 * `requestFullscreen` and then declines it for anything that is not a video
 * element. So a refusal falls back to filling the page instead, and the caller
 * only ever has to ask whether the call is expanded.
 *
 * The element passed in is the whole call — picture and controls — never just
 * the video: a fullscreen call you cannot stop is not a fullscreen call.
 */
export type FullscreenState = {
  /** True when the call should fill the screen, however that is achieved. */
  expanded: boolean;
  /** True when the browser is really in fullscreen, with no page chrome at all. */
  native: boolean;
  toggle: () => void;
  /** Give the screen back. Used when the call ends, whatever the mode. */
  collapse: () => void;
};

/** How long a request is given to take effect before the fallback steps in. */
const REQUEST_VERIFY_MS = 500;

export function useFullscreen(target: RefObject<HTMLElement | null>): FullscreenState {
  const [native, setNative] = useState(false);
  const [filled, setFilled] = useState(false);

  // A pending "did that actually work" check, so an unmount cannot leave a timer
  // behind that flips state on a call that is already over.
  const verifyRef = useRef<number | null>(null);

  const clearVerify = useCallback(() => {
    if (verifyRef.current !== null) {
      window.clearTimeout(verifyRef.current);
      verifyRef.current = null;
    }
  }, []);

  // The browser owns the real fullscreen state: Escape, a phone's back gesture
  // and the browser's own button all change it without telling this code.
  useEffect(() => {
    const handleChange = () => {
      const element = document.fullscreenElement;
      setNative(element !== null);

      if (element !== null) {
        clearVerify();
      } else {
        // Leaving fullscreen by any route leaves the page as it was.
        setFilled(false);
      }
    };

    document.addEventListener("fullscreenchange", handleChange);
    return () => document.removeEventListener("fullscreenchange", handleChange);
  }, [clearVerify]);

  // Filling the page means owning the viewport for a while: the page behind must
  // not scroll, and Escape should undo it the way it undoes real fullscreen.
  useEffect(() => {
    if (!filled) {
      return;
    }

    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }
      // A dialog on top of the call owns Escape while it is open.
      if (event.target instanceof Element && event.target.closest('[role="dialog"]')) {
        return;
      }
      setFilled(false);
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [filled]);

  // A call that ends, or a route change, must not leave the browser fullscreen.
  useEffect(() => {
    return () => {
      clearVerify();
      if (document.fullscreenElement !== null) {
        void document.exitFullscreen().catch(() => undefined);
      }
    };
  }, [clearVerify]);

  const collapse = useCallback(() => {
    setFilled(false);
    if (document.fullscreenElement !== null) {
      void document.exitFullscreen().catch(() => undefined);
    }
  }, []);

  const toggle = useCallback(() => {
    const element = target.current;
    if (!element) {
      return;
    }

    if (document.fullscreenElement !== null || filled) {
      collapse();
      return;
    }

    // The cheapest signal that this browser will actually do it.
    if (!document.fullscreenEnabled) {
      setFilled(true);
      return;
    }

    const request = element.requestFullscreen();
    if (request && typeof request.catch === "function") {
      request.catch(() => setFilled(true));
    }

    // A resolved promise is not proof either, so the entry is verified: if the
    // element never became the fullscreen element, the page fills instead.
    verifyRef.current = window.setTimeout(() => {
      verifyRef.current = null;
      if (document.fullscreenElement === null) {
        setFilled(true);
      }
    }, REQUEST_VERIFY_MS);
  }, [collapse, filled, target]);

  return { expanded: native || filled, native, toggle, collapse };
}
