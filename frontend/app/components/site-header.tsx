import { useEffect, useState } from "react";
import { Menu, X } from "lucide-react";
import { Link, useLocation } from "react-router";

import { ROUTES } from "../../lib/config";
import { cn } from "../../lib/utils";
import { LiveDot } from "./spinner";
import { Button } from "./ui/button";
import { Wordmark } from "./wordmark";

const LANDING_LINKS = [
  { to: ROUTES.guidelines, label: "Guidelines" },
  { to: ROUTES.privacy, label: "Privacy" },
  { to: ROUTES.about, label: "About" },
];

/**
 * The top bar.
 *
 * On the paper pages it is a translucent strip over the content with a single
 * primary action. On the call screen it turns dark and swaps navigation for the
 * one thing that matters mid-call: safety. Nothing here ever leads away from a
 * live call.
 */
export function SiteHeader({
  variant = "landing",
  onSafetyClick,
  inCall = false,
}: {
  variant?: "landing" | "call";
  onSafetyClick?: () => void;
  inCall?: boolean;
}) {
  const location = useLocation();
  const [menuOpen, setMenuOpen] = useState(false);

  // A route change always closes the mobile panel.
  useEffect(() => setMenuOpen(false), [location.pathname]);

  useEffect(() => {
    if (!menuOpen) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMenuOpen(false);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [menuOpen]);

  const onStage = variant === "call";

  return (
    <header
      className={cn(
        "sticky top-0 z-40",
        onStage
          ? "border-b border-stage-line bg-stage/85 backdrop-blur-md"
          : "border-b border-hairline/80 bg-paper/85 backdrop-blur-md",
      )}
    >
      <div className="shell flex h-16 items-center justify-between gap-4">
        <div className="flex min-w-0 items-center gap-4">
          <Link
            to={ROUTES.home}
            aria-label="Strangerly home"
            className="rounded-control transition-opacity duration-200 hover:opacity-80"
          >
            <Wordmark tone={onStage ? "stage" : "ink"} />
          </Link>

          {onStage ? (
            <span className="hidden items-center gap-2 rounded-chip border border-stage-line px-2.5 py-1 sm:inline-flex">
              <LiveDot tone={inCall ? "signal" : "success"} />
              <span className="mono-label text-stage-muted">{inCall ? "In call" : "Ready"}</span>
            </span>
          ) : null}
        </div>

        {onStage ? (
          <Button type="button" variant="stage" size="sm" onClick={onSafetyClick}>
            Safety &amp; reporting
          </Button>
        ) : (
          <>
            <nav aria-label="Site" className="hidden items-center gap-7 md:flex">
              {LANDING_LINKS.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  aria-current={location.pathname === link.to ? "page" : undefined}
                  className={cn(
                    "link-underline text-small transition-colors duration-200",
                    location.pathname === link.to ? "text-ink" : "text-ink-soft hover:text-ink",
                  )}
                >
                  {link.label}
                </Link>
              ))}
            </nav>

            <div className="flex items-center gap-2">
              <Button asChild className="hidden sm:inline-flex">
                <a href="#start">Start a call</a>
              </Button>

              <button
                type="button"
                aria-expanded={menuOpen}
                aria-controls="site-menu"
                aria-label={menuOpen ? "Close menu" : "Open menu"}
                onClick={() => setMenuOpen((open) => !open)}
                className="grid size-11 place-items-center rounded-control text-ink transition-colors duration-200 hover:bg-clay md:hidden"
              >
                {menuOpen ? (
                  <X className="size-5" aria-hidden="true" />
                ) : (
                  <Menu className="size-5" aria-hidden="true" />
                )}
              </button>
            </div>
          </>
        )}
      </div>

      {menuOpen && !onStage ? (
        <div id="site-menu" className="border-t border-hairline bg-paper md:hidden">
          <nav aria-label="Site" className="shell flex flex-col py-2">
            {LANDING_LINKS.map((link) => (
              <Link
                key={link.to}
                to={link.to}
                className="border-b border-hairline/70 py-3.5 text-body-lg text-ink last:border-0"
              >
                {link.label}
              </Link>
            ))}
            <Button asChild size="lg" className="mt-3 mb-2">
              <a href="#start">Start a call</a>
            </Button>
          </nav>
        </div>
      ) : null}
    </header>
  );
}
