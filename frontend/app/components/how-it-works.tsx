import { Reveal } from "./reveal";
import { StagePreview } from "./stage-preview";

const STEPS = [
  {
    number: "01",
    title: "Choose a name",
    body: "A label for this visit, not an account. Nothing to verify, nothing to remember, nothing that has to be unique.",
  },
  {
    number: "02",
    title: "Allow your camera",
    body: "Your browser asks once, and only after you press the button. The preview shows exactly what the other person will see.",
  },
  {
    number: "03",
    title: "Meet one person",
    body: "The queue is first in, first out. You are paired with a single waiting visitor and the video travels straight between the two of you.",
  },
];

/**
 * The dark band: how a match happens, next to the stage it happens on.
 *
 * This is the page's second movement — a different tone, a different rhythm
 * (numbered list beside a product surface), and the only place the call UI is
 * shown on the landing page.
 */
export function HowItWorks() {
  return (
    <section id="how" className="grain relative scroll-mt-20 overflow-hidden bg-stage text-stage-text">
      <div className="grain-overlay" aria-hidden="true" />

      <div className="shell relative band">
        <div className="grid gap-12 lg:grid-cols-[1.05fr_1.15fr] lg:items-start lg:gap-16">
          <div>
            <Reveal>
              <p className="mono-label text-stage-muted">How matching works</p>
              <h2 className="mt-5 max-w-lg font-display text-display-lg">
                Three steps, then a stranger.
              </h2>
              <p className="measure mt-5 text-body-lg text-stage-muted">
                No feed to scroll, no profiles to compare. One queue, one partner, and a call that
                ends the moment either of you wants it to.
              </p>
            </Reveal>

            <ol className="mt-10 flex flex-col">
              {STEPS.map((step, index) => (
                <Reveal as="li" key={step.number} delay={80 + index * 70} className="border-t border-stage-line py-7 first:border-t-0 first:pt-0">
                  <div className="flex gap-6">
                    <span className="font-display text-[1.75rem] leading-none text-signal-bright">
                      {step.number}
                    </span>
                    <div className="min-w-0">
                      <h3 className="font-display text-display-sm text-stage-text">{step.title}</h3>
                      <p className="measure mt-2 text-small text-stage-muted">{step.body}</p>
                    </div>
                  </div>
                </Reveal>
              ))}
            </ol>
          </div>

          <Reveal delay={140}>
            <StagePreview />
          </Reveal>
        </div>
      </div>
    </section>
  );
}
