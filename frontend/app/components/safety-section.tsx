import { Link } from "react-router";

import { ROUTES } from "../../lib/config";
import { Reveal } from "./reveal";

const POINTS = [
  {
    index: "01",
    title: "Stop works immediately",
    body: "The five second rule applies to Skip only. Stop, muting, turning your camera off, reporting, and closing the tab all work in the first second of a call.",
  },
  {
    index: "02",
    title: "Report during the call",
    body: "The report button is available the whole time you are connected, and for a moment after the call ends. A report stores the reason and your note — never the video, never your name.",
  },
  {
    index: "03",
    title: "Blocking is local",
    body: "With no accounts there is nothing to attach a block list to, so a person can be matched with you again in a later visit. This version says that plainly instead of implying otherwise.",
  },
  {
    index: "04",
    title: "What is not verified",
    body: "No identity, age, or content check happens before a call. Reports are read by a person, not in real time, and Strangerly is not a crisis service. If someone is in danger, contact local emergency services.",
  },
];

/**
 * The clay band: the safety section.
 *
 * Different tone, different structure — a sticky heading beside a ruled list
 * instead of a grid of cards. Safety copy that looks like marketing copy does
 * not get read.
 */
export function SafetySection() {
  return (
    <section id="safety" className="scroll-mt-20 border-y border-hairline bg-clay">
      <div className="shell band">
        <div className="grid gap-12 lg:grid-cols-[0.85fr_1.15fr] lg:gap-20">
          <div className="lg:sticky lg:top-28 lg:self-start">
            <Reveal>
              <p className="mono-label text-ink-muted">Safety</p>
              <h2 className="mt-5 font-display text-display-md text-ink">
                If something goes wrong
              </h2>
              <p className="measure mt-5 text-body text-ink-soft">
                You are in control of every call. Nothing is buried behind a menu, and nothing is
                locked while you wait.
              </p>
              <p className="mt-6 flex flex-wrap gap-x-6 gap-y-2 text-small">
                <Link to={ROUTES.guidelines} className="link-underline text-ink">
                  Community guidelines
                </Link>
                <Link to={ROUTES.privacy} className="link-underline text-ink">
                  Privacy notes
                </Link>
              </p>

              {/* What actually happens to a report — the question people ask
                  right after being told to press the button. */}
              <div className="mt-10 rounded-surface border border-hairline-strong bg-surface/60 p-5">
                <p className="mono-label text-ink-muted">After you report</p>
                <ol className="mt-4 flex flex-col gap-3">
                  {[
                    "The reason, your note, the match, and the time are stored.",
                    "A person reads it. There is no automated system acting on it.",
                    "No video or audio exists to attach, because none is recorded.",
                  ].map((line, index) => (
                    <li key={line} className="flex gap-3 text-small text-ink-soft">
                      <span className="mono-label pt-0.5 text-signal">
                        {String(index + 1).padStart(2, "0")}
                      </span>
                      <span>{line}</span>
                    </li>
                  ))}
                </ol>
              </div>
            </Reveal>
          </div>

          <ul className="flex flex-col border-t border-hairline-strong">
            {POINTS.map((point, index) => (
              <Reveal
                as="li"
                key={point.index}
                delay={index * 60}
                className="grid gap-3 border-b border-hairline-strong py-7 sm:grid-cols-[3.5rem_1fr] sm:gap-6"
              >
                <span className="mono-label text-signal">{point.index}</span>
                <div>
                  <h3 className="font-display text-display-sm text-ink">{point.title}</h3>
                  <p className="measure mt-2 text-small text-ink-soft">{point.body}</p>
                </div>
              </Reveal>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}
