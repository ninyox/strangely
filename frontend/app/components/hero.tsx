import { ArrowRight, ShieldCheck, Sparkles, Users } from "lucide-react";

import { LivePanel } from "./live-panel";
import { LiveDot } from "./spinner";
import { Reveal } from "./reveal";
import { Button } from "./ui/button";

const PROMISES = [
  { icon: Sparkles, text: "No account" },
  { icon: Users, text: "One stranger at a time" },
  { icon: ShieldCheck, text: "Nothing recorded" },
];

/**
 * The hero.
 *
 * An asymmetric editorial composition: the promise on the left, the product
 * itself on the right, and a quiet signal motif behind both. The headline is
 * the largest type on the site and the only place the accent touches the copy,
 * because "someone new" is the entire proposition.
 */
export function Hero() {
  return (
    <section className="relative overflow-hidden pb-14 pt-12 sm:pt-16 lg:pb-24 lg:pt-20">
      {/* Signal motif: three hairline rings bleeding off the right edge behind
          the panel. Decoration with a job — it says "a connection is forming". */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -right-[22rem] top-[-14rem] hidden size-[54rem] lg:block"
      >
        {[0, 1, 2].map((ring) => (
          <span
            key={ring}
            className="absolute inset-0 rounded-full border border-hairline-strong"
            style={{
              transform: `scale(${1 - ring * 0.22})`,
              opacity: 0.5 - ring * 0.12,
            }}
          />
        ))}
      </div>

      <div className="shell relative">
        <div className="grid items-center gap-12 lg:grid-cols-[1.08fr_0.92fr] lg:gap-16">
          <div className="flex flex-col">
            <Reveal>
              <span className="inline-flex items-center gap-2.5 rounded-chip border border-hairline-strong bg-surface px-3 py-1.5">
                <LiveDot />
                <span className="mono-label text-ink-soft">Live · one-to-one · no account</span>
              </span>
            </Reveal>

            <Reveal delay={60}>
              <h1 className="mt-7 text-balance font-display text-display-xl text-ink">
                Meet <span className="text-signal">someone new</span>.
              </h1>
            </Reveal>

            <Reveal delay={120}>
              <p className="mt-6 max-w-[44ch] text-lead text-ink-soft">
                Turn on your camera, press start, and you are paired with one other person for a real
                conversation. No profile, nothing to install, and nothing kept when you close the tab.
              </p>
            </Reveal>

            <Reveal delay={180}>
              <div className="mt-9 flex flex-wrap items-center gap-x-7 gap-y-3">
                <Button asChild size="xl">
                  <a href="#start">
                    Start matching
                    <ArrowRight aria-hidden="true" />
                  </a>
                </Button>
                <a
                  href="#how"
                  className="link-underline text-body font-medium text-ink transition-colors duration-200"
                >
                  See how matching works
                </a>
              </div>
            </Reveal>

            <Reveal delay={240}>
              <ul className="mt-10 flex flex-wrap items-center gap-x-6 gap-y-3 border-t border-hairline pt-6">
                {PROMISES.map((promise) => (
                  <li key={promise.text} className="flex items-center gap-2 text-small text-ink-soft">
                    <promise.icon className="size-4 text-signal" aria-hidden="true" />
                    {promise.text}
                  </li>
                ))}
              </ul>
            </Reveal>
          </div>

          <Reveal delay={140} className="relative mx-auto w-full max-w-[30rem] lg:mx-0 lg:max-w-none">
            <LivePanel />
          </Reveal>
        </div>
      </div>
    </section>
  );
}
