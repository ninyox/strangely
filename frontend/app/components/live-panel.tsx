import { useState } from "react";
import { ArrowRight } from "lucide-react";
import { Link, useNavigate } from "react-router";

import { useCall } from "../../lib/call";
import { ROUTES } from "../../lib/config";
import { useSession } from "../../lib/session";
import { normalizeUsername, suggestUsername, usernameProblem } from "../../lib/username";
import { CameraPreview } from "./camera-preview";
import { Notice } from "./notice";
import { Spinner } from "./spinner";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Surface } from "./ui/surface";

/**
 * The live panel.
 *
 * This is the product, presented as itself rather than as a signup form: a
 * labelled panel with a preview monitor, the name you will be known by, and one
 * action. The camera is requested only from the button inside the monitor, and
 * matching is requested only from the submit button.
 */
export function LivePanel() {
  const session = useSession();
  const call = useCall();
  const navigate = useNavigate();

  const [formProblem, setFormProblem] = useState<string | null>(null);

  const fieldProblem = usernameProblem(session.username);
  const problem = formProblem ?? fieldProblem;
  const canStart =
    session.status === "ready" && session.username.trim() !== "" && fieldProblem === null;

  const handleStart = (event: React.FormEvent) => {
    event.preventDefault();

    const result = normalizeUsername(session.username);
    if (!result.ok) {
      setFormProblem(result.message);
      return;
    }

    setFormProblem(null);
    session.setUsername(result.value);
    // The call screen owns everything that follows: media permission, the
    // queue, the match, and every state after it.
    navigate(ROUTES.chat);
  };

  return (
    <Surface id="start" className="scroll-mt-24 overflow-hidden shadow-lift">
      <div className="flex items-center justify-between gap-3 border-b border-hairline px-5 py-3 sm:px-6">
        <span className="mono-label text-ink-muted">New session</span>
        <span className="mono-label text-ink-muted">No account needed</span>
      </div>

      <div className="p-5 sm:p-6">
        {session.status === "error" ? (
          <Notice tone="error" className="mb-5" title="Could not start a session">
            <p>{session.error}</p>
            <Button type="button" variant="outline" size="sm" className="mt-3" onClick={session.retry}>
              Try again
            </Button>
          </Notice>
        ) : null}

        <form onSubmit={handleStart} className="flex flex-col gap-4" noValidate>
          <div className="flex flex-col gap-2">
            <div className="flex items-baseline justify-between gap-3">
              <label htmlFor="username" className="text-small font-medium text-ink">
                Your name for this visit
              </label>
              <button
                type="button"
                onClick={() => {
                  setFormProblem(null);
                  session.setUsername(suggestUsername());
                }}
                disabled={session.status !== "ready"}
                className="mono-label text-ink-muted underline decoration-hairline-strong underline-offset-4 transition-colors duration-200 hover:text-signal disabled:opacity-50"
              >
                Suggest one
              </button>
            </div>

            <Input
              id="username"
              name="username"
              value={session.username}
              onChange={(event) => {
                setFormProblem(null);
                session.setUsername(event.target.value);
              }}
              placeholder="e.g. Ada"
              autoComplete="off"
              maxLength={40}
              aria-invalid={problem ? true : undefined}
              aria-describedby={problem ? "username-problem" : "username-hint"}
              disabled={session.status !== "ready"}
            />

            {problem ? (
              <p id="username-problem" className="text-caption text-danger">
                {problem}
              </p>
            ) : (
              <p id="username-hint" className="text-caption text-ink-muted">
                3–20 characters. A label for this visit, not an identity.
              </p>
            )}
          </div>

          <CameraPreview />

          <Button type="submit" size="xl" className="w-full" disabled={!canStart || call.busy}>
            {session.status === "loading" ? (
              <Spinner tone="stage" label="Preparing your session…" />
            ) : (
              <>
                Start matching
                <ArrowRight aria-hidden="true" />
              </>
            )}
          </Button>

          <p className="text-caption leading-relaxed text-ink-muted">
            By starting you agree to the{" "}
            <Link
              to={ROUTES.guidelines}
              className="link-underline text-ink-soft transition-colors duration-200 hover:text-ink"
            >
              community guidelines
            </Link>
            : keep it clothed, keep it kind, be 18 or over. What you show is seen by one stranger, in
            real time, and is never recorded.
          </p>
        </form>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-hairline bg-clay/50 px-5 py-3 text-caption text-ink-muted sm:px-6">
        <span>No recording. No chat history. No profile.</span>
        <Link
          to={ROUTES.privacy}
          className="link-underline transition-colors duration-200 hover:text-ink"
        >
          How your data is handled
        </Link>
      </div>
    </Surface>
  );
}
