import { Flag, PhoneOff } from "lucide-react";
import { Link } from "react-router";

import { ROUTES } from "../../lib/config";
import { Button } from "./ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "./ui/dialog";

const POINTS = [
  {
    title: "Stop ends the call instantly",
    body: "It works during the first five seconds, while the video is still connecting, and at any other moment.",
  },
  {
    title: "Report the call if something was wrong",
    body: "Reports store no recording, no name, and no details about the other person. A person reviews them.",
  },
  {
    title: "Close the tab if you need to",
    body: "Nothing is locked. The other person is told you left instead of being left watching a frozen picture.",
  },
  {
    title: "What is not verified",
    body: "Nobody is identity or age checked, and no content is filtered before it reaches you.",
  },
];

/**
 * The safety panel reachable from the call screen.
 *
 * It exists so a visitor in an uncomfortable call does not have to navigate away
 * (which would end the call) to find out what their options are. Both actions
 * here are immediate.
 */
export function SafetyDialog({
  open,
  onOpenChange,
  onReport,
  onStop,
  canReport,
  canStop,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onReport: () => void;
  onStop: () => void;
  canReport: boolean;
  canStop: boolean;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogTitle>Safety and reporting</DialogTitle>
        <DialogDescription>
          You can leave or report at any moment. The five second rule only ever applies to Skip.
        </DialogDescription>

        <ul className="mt-6 flex flex-col border-t border-hairline">
          {POINTS.map((point) => (
            <li key={point.title} className="border-b border-hairline py-4">
              <p className="text-small font-medium text-ink">{point.title}</p>
              <p className="mt-1 text-small text-ink-soft">{point.body}</p>
            </li>
          ))}
        </ul>

        <DialogFooter>
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Back to the call
          </Button>
          {canStop ? (
            <Button type="button" variant="outline" onClick={onStop}>
              <PhoneOff aria-hidden="true" />
              Stop the call
            </Button>
          ) : null}
          <Button type="button" onClick={onReport} disabled={!canReport}>
            <Flag aria-hidden="true" />
            Report
          </Button>
        </DialogFooter>

        <p className="mt-5 text-caption text-ink-muted">
          The full{" "}
          <Link to={ROUTES.guidelines} className="link-underline text-ink-soft">
            community guidelines
          </Link>{" "}
          and{" "}
          <Link to={ROUTES.privacy} className="link-underline text-ink-soft">
            privacy notes
          </Link>{" "}
          live on their own pages. Opening them ends the call, which is why the essentials are
          repeated here.
        </p>
      </DialogContent>
    </Dialog>
  );
}
