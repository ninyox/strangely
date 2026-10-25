import { Link } from "react-router";

import { SiteFooter } from "../components/site-footer";
import { SiteHeader } from "../components/site-header";
import { SignalRings } from "../components/spinner";
import { Button } from "../components/ui/button";

export default function NotFoundRoute() {
  return (
    <div className="flex min-h-screen flex-col">
      <SiteHeader />

      <main className="flex-1">
        <div className="shell flex flex-col items-start py-20 lg:py-32">
          <p className="mono-label text-ink-muted">Error 404</p>

          <h1 className="mt-6 max-w-2xl font-display text-display-lg text-ink">
            That page is not here.
          </h1>

          <p className="measure mt-6 text-lead text-ink-soft">
            The link may be old, or the address may have a typo. The way back is the home page, where
            you can be talking to someone in about ten seconds.
          </p>

          <div className="mt-9 flex flex-wrap items-center gap-4">
            <Button asChild size="lg">
              <Link to="/">Go to the home page</Link>
            </Button>
            <Link to="/guidelines" className="link-underline text-body font-medium text-ink">
              Read the guidelines
            </Link>
          </div>

          <div className="relative mt-16 grid size-40 place-items-center" aria-hidden="true">
            <SignalRings className="size-40" tone="paper" animate={false} />
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  );
}
