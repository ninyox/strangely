import { useEffect, useState } from "react";
import { Flag } from "lucide-react";

import { apiErrorMessage } from "../../lib/api";
import { REPORT_REASONS } from "../../lib/types";
import { cn } from "../../lib/utils";
import { Notice } from "./notice";
import { Button } from "./ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "./ui/dialog";

/**
 * The report flow.
 *
 * Available for the whole call, including the first five seconds, and for a
 * moment after it ends. Reasons are real radio inputs styled as rows — keyboard
 * arrow keys work, focus is visible, and the copy says exactly what will happen
 * next (a person reads it; nothing is recorded).
 */
export function ReportDialog({
  open,
  onOpenChange,
  onSubmit,
  partnerUsername,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (reason: string, note: string) => Promise<void>;
  partnerUsername: string | null;
}) {
  const [reason, setReason] = useState("");
  const [note, setNote] = useState("");
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setReason("");
      setNote("");
      setSent(false);
      setFailure(null);
    }
  }, [open]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (reason === "") {
      setFailure("Choose a reason so the report can be acted on.");
      return;
    }

    setSending(true);
    setFailure(null);
    try {
      await onSubmit(reason, note);
      setSent(true);
    } catch (cause) {
      setFailure(apiErrorMessage(cause, "Could not send the report. Please try again."));
    } finally {
      setSending(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogTitle>Report this call</DialogTitle>
        <DialogDescription>
          {partnerUsername ? `Tell us what happened with ${partnerUsername}.` : "Tell us what happened."}{" "}
          A person reads every report. Strangerly stores no video or audio, so a report cannot include
          a recording.
        </DialogDescription>

        {sent ? (
          <>
            <Notice tone="success" className="mt-6" title="Report received">
              Thank you — it has been logged. You can end the call at any time with Stop.
            </Notice>
            <DialogFooter>
              <Button type="button" onClick={() => onOpenChange(false)}>
                Close
              </Button>
            </DialogFooter>
          </>
        ) : (
          <form onSubmit={handleSubmit} className="mt-6">
            <fieldset className="flex flex-col gap-2">
              <legend className="text-small font-medium text-ink">What happened?</legend>
              {REPORT_REASONS.map((option) => (
                <label
                  key={option.value}
                  className={cn(
                    "flex cursor-pointer items-center gap-3 rounded-control border px-3.5 py-3 text-small transition-colors duration-200 ease-settle",
                    reason === option.value
                      ? "border-signal bg-signal-tint text-ink"
                      : "border-hairline-strong text-ink-soft hover:border-ink-muted hover:bg-clay",
                  )}
                >
                  <input
                    type="radio"
                    name="report-reason"
                    value={option.value}
                    checked={reason === option.value}
                    onChange={(event) => setReason(event.target.value)}
                    className="size-4 accent-[var(--color-signal)]"
                  />
                  {option.label}
                </label>
              ))}
            </fieldset>

            <label htmlFor="report-note" className="mt-6 block text-small font-medium text-ink">
              Anything else? (optional)
            </label>
            <textarea
              id="report-note"
              value={note}
              onChange={(event) => setNote(event.target.value)}
              rows={3}
              maxLength={500}
              placeholder="A short description helps, but only if you want to write one."
              className="mt-2 w-full rounded-control border border-hairline-strong bg-surface px-3.5 py-3 text-small text-ink transition-colors duration-200 placeholder:text-ink-muted hover:border-ink-muted focus:border-signal focus:outline-none"
            />

            {failure ? (
              <Notice tone="error" className="mt-4">
                {failure}
              </Notice>
            ) : null}

            <DialogFooter>
              <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={sending}>
                <Flag aria-hidden="true" />
                {sending ? "Sending…" : "Send report"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
