import { ClosingCta } from "../components/closing-cta";
import { Hero } from "../components/hero";
import { HowItWorks } from "../components/how-it-works";
import { SafetySection } from "../components/safety-section";
import { SiteFooter } from "../components/site-footer";
import { SiteHeader } from "../components/site-header";

/**
 * The landing page, in four movements: paper hero, dark stage, clay safety,
 * signal closing. No two adjacent sections share a tone or a layout, which is
 * what gives the page its rhythm.
 */
export default function HomeRoute() {
  return (
    <div className="flex min-h-screen flex-col">
      <SiteHeader />

      <main className="flex-1">
        <Hero />
        <HowItWorks />
        <SafetySection />
        <ClosingCta />
      </main>

      <SiteFooter />
    </div>
  );
}
