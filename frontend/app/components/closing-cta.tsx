import { ArrowRight } from "lucide-react";
import { Link } from "react-router";

import { ROUTES } from "../../lib/config";
import { Reveal } from "./reveal";
import { Button } from "./ui/button";

const FACTS = [
  "Free, and no account",
  "Camera off until you allow it",
  "Stop and report always available",
];

/**
 * The closing band.
 *
 * A single saturated block at the end of the page: the conversion moment,
 * unmissable, and the only place the accent covers a whole section. All text is
 * solid white on the signal red (4.83:1), so hierarchy comes from size and
 * weight rather than from transparency.
 */
export function ClosingCta() {
  return (
    <section className="grain relative overflow-hidden bg-signal text-white">
      <div className="grain-overlay" aria-hidden="true" />

      <div className="shell relative band">
        <div className="grid gap-10 lg:grid-cols-[1.15fr_0.85fr] lg:items-end lg:gap-16">
          <Reveal>
            <h2 className="max-w-2xl font-display text-display-lg">Ready to meet someone?</h2>
            <p className="measure mt-5 text-lead">
              One click opens the camera and puts you in the queue. The next person to arrive is the
              person you meet.
            </p>

            <div className="mt-9 flex flex-wrap items-center gap-4">
              <Button asChild variant="inverse" size="xl">
                <a href="#start">
                  Start matching
                  <ArrowRight aria-hidden="true" />
                </a>
              </Button>
              <Button asChild variant="ghost" size="xl" className="text-white hover:bg-white/12 hover:text-white">
                <Link to={ROUTES.guidelines}>Read the rules first</Link>
              </Button>
            </div>
          </Reveal>

          <Reveal delay={120}>
            <ul className="flex flex-col border-t border-white/30">
              {FACTS.map((fact) => (
                <li key={fact} className="border-b border-white/30 py-4 text-small">
                  {fact}
                </li>
              ))}
            </ul>
            <p className="mono-label mt-6">Adults only · Not moderated in real time</p>
          </Reveal>
        </div>
      </div>
    </section>
  );
}
