import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";

import { CallControls } from "../components/call-controls";
import { CallStage } from "../components/call-stage";
import { ReportDialog } from "../components/report-dialog";
import { SafetyDialog } from "../components/safety-dialog";
import { SiteFooter } from "../components/site-footer";
import { SiteHeader } from "../components/site-header";
import {
  CallWarning,
  EndedPanel,
  FailedPanel,
  PreparingPanel,
  SearchingPanel,
} from "../components/state-panels";
import { useCall } from "../../lib/call";
import { ROUTES } from "../../lib/config";
import { useFullscreen } from "../../lib/fullscreen";
import { useMedia } from "../../lib/media";
import { useSession } from "../../lib/session";
import { cn } from "../../lib/utils";

/**
 * The call screen.
 *
 * One route renders every state of the conversation — waiting, connecting,
 * connected, ended, failed — because they are all the same visit, and moving
 * between routes would tear down a live peer connection. It is a dark stage on
 * purpose: this screen should feel like a room, not like a page.
 */
export default function ChatRoute() {
  const session = useSession();
  const call = useCall();
  const media = useMedia();
  const navigate = useNavigate();

  // The flow is started exactly once per visit to this route.
  const started = useRef(false);

  // Full screen is a property of the call, not of the route: the frame holds the
  // picture and the controls together so neither can be lost.
  const callFrameRef = useRef<HTMLDivElement | null>(null);
  const { expanded, toggle: toggleFullscreen, collapse: collapseFullscreen } =
    useFullscreen(callFrameRef);

  const inCall = call.phase === "connecting" || call.phase === "connected";

  // A call that ends gives the screen back, in either mode: the frame is gone,
  // so a filled page would otherwise stay locked behind it.
  useEffect(() => {
    if (!inCall) {
      collapseFullscreen();
    }
  }, [inCall, collapseFullscreen]);

  const [reportOpen, setReportOpen] = useState(false);
  const [safetyOpen, setSafetyOpen] = useState(false);

  const username = session.readyUsername();

  useEffect(() => {
    // Wait for the anonymous session before deciding anything: on a hard reload
    // of /chat the cookie check is still in flight.
    if (session.status === "loading" || started.current) {
      return;
    }

    started.current = true;

    if (!username) {
      // Arrived here without a name: the landing page is where names are set.
      navigate(ROUTES.home, { replace: true });
      return;
    }

    if (call.phase === "idle") {
      void call.startMatching(username);
    }
  }, [session.status, username, navigate, call]);

  const leave = () => {
    call.reset();
    navigate(ROUTES.home);
  };

  return (
    <div className="flex min-h-screen flex-col bg-stage text-stage-text">
      <SiteHeader variant="call" inCall={inCall} onSafetyClick={() => setSafetyOpen(true)} />

      <main className="mx-auto w-full max-w-4xl flex-1 px-4 pb-4 pt-5 sm:px-6 sm:pb-8 sm:pt-8">
        <h1 className="sr-only">Your Strangerly call</h1>

        <div className="flex flex-col gap-4">
          {call.error && inCall ? <CallWarning message={call.error} /> : null}

          {call.phase === "queued" ? (
            <SearchingPanel
              queuePosition={0}
              busy={call.busy}
              onCancel={() => {
                void call.cancelMatching().then(() => navigate(ROUTES.home));
              }}
            />
          ) : null}

          {inCall ? (
            <div
              ref={callFrameRef}
              className={cn(
                "flex flex-col gap-4",
                // The full screen target is the whole call, controls included:
                // filling the screen must never take Stop away.
                expanded && "fixed inset-0 z-40 gap-0 bg-stage px-0",
              )}
            >
              <CallStage
                remoteStream={call.remoteStream}
                localStream={media.stream}
                partnerUsername={call.partnerUsername}
                phase={call.phase === "connected" ? "connected" : "connecting"}
                connectionState={call.connectionState}
                signalingConnected={call.signalingConnected}
                peerUnreachable={call.peerUnreachable}
                expanded={expanded}
                onToggleFullscreen={toggleFullscreen}
              />

              <CallControls
                remainingMs={call.remainingMs}
                skipLockMs={call.skipLockMs}
                busy={call.busy}
                expanded={expanded}
                onSkip={() => void call.skip()}
                onStop={() => void call.stop()}
                onReport={() => setReportOpen(true)}
              />
            </div>
          ) : null}

          {call.phase === "ended" ? (
            <EndedPanel
              reason={call.endedReason}
              message={call.error}
              busy={call.busy}
              onFindSomeoneNew={() => void call.findSomeoneNew(username ?? undefined)}
              onLeave={leave}
              onReport={() => setReportOpen(true)}
            />
          ) : null}

          {call.phase === "idle" ? (
            call.error ? (
              <FailedPanel
                message={call.error}
                busy={call.busy}
                onRetry={() => {
                  if (username) {
                    void call.startMatching(username);
                  } else {
                    navigate(ROUTES.home);
                  }
                }}
                onLeave={leave}
              />
            ) : (
              <PreparingPanel
                message={
                  media.status === "requesting" ? "Waiting for camera permission" : undefined
                }
                step={
                  media.status === "requesting"
                    ? "Your browser is asking · nothing is sent until you allow it"
                    : "Opening the connection"
                }
              />
            )
          ) : null}
        </div>
      </main>

      <SiteFooter variant="call" />

      <ReportDialog
        open={reportOpen}
        onOpenChange={setReportOpen}
        partnerUsername={call.partnerUsername}
        onSubmit={async (reason, note) => {
          await call.report(reason, note);
        }}
      />

      <SafetyDialog
        open={safetyOpen}
        onOpenChange={setSafetyOpen}
        onReport={() => {
          setSafetyOpen(false);
          setReportOpen(true);
        }}
        onStop={() => {
          setSafetyOpen(false);
          void call.stop();
        }}
        canReport={inCall || call.phase === "ended"}
        canStop={inCall}
      />
    </div>
  );
}
