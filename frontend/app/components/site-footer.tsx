import { Link } from "react-router";

import { ROUTES } from "../../lib/config";
import { LiveDot } from "./spinner";
import { Mark } from "./wordmark";

const COLUMNS = [
  {
    title: "Use it",
    links: [
      { to: ROUTES.home, label: "Start a call" },
      { to: ROUTES.guidelines, label: "Community guidelines" },
      { to: ROUTES.about, label: "How matching works" },
    ],
  },
  {
    title: "Trust",
    links: [
      { to: ROUTES.privacy, label: "Privacy" },
      { to: `${ROUTES.privacy}#never`, label: "What we never store" },
      { to: `${ROUTES.about}#what-it-is-not`, label: "Known limitations" },
    ],
  },
];

/**
 * The footer.
 *
 * Three columns and a closing line that repeats the two facts a stranger-video
 * product should never hide: nothing is recorded, and nothing is moderated in
 * real time.
 */
export function SiteFooter({ variant = "landing" }: { variant?: "landing" | "call" }) {
  const onStage = variant === "call";

  return (
    <footer
      className={
        onStage
          ? "border-t border-stage-line bg-stage"
          : "mt-auto border-t border-hairline bg-paper"
      }
    >
      <div className="shell py-12 lg:py-16">
        <div className="grid gap-10 lg:grid-cols-[1.4fr_1fr_1fr]">
          <div className="flex max-w-sm flex-col gap-4">
            <div className="flex items-center gap-2.5">
              <Mark tone={onStage ? "stage" : "ink"} className="size-5" />
              <span
                className={
                  onStage
                    ? "font-display text-[1.1875rem] font-medium text-stage-text"
                    : "font-display text-[1.1875rem] font-medium text-ink"
                }
              >
                Strangerly
              </span>
            </div>
            <p className={onStage ? "text-small text-stage-muted" : "text-small text-ink-soft"}>
              One click away from someone new. A single conversation with one stranger, and nothing
              kept afterwards.
            </p>
            <p
              className={
                onStage
                  ? "mono-label flex items-center gap-2 text-stage-muted"
                  : "mono-label flex items-center gap-2 text-ink-muted"
              }
            >
              <LiveDot tone="success" />
              No recording · No accounts
            </p>
          </div>

          {COLUMNS.map((column) => (
            <nav key={column.title} aria-label={column.title} className="flex flex-col gap-3.5">
              <p className={onStage ? "mono-label text-stage-muted" : "mono-label text-ink-muted"}>
                {column.title}
              </p>
              <ul className="flex flex-col gap-2.5">
                {column.links.map((link) => (
                  <li key={`${column.title}-${link.label}`}>
                    <Link
                      to={link.to}
                      className={
                        onStage
                          ? "link-underline text-small text-stage-text/85 transition-colors duration-200 hover:text-stage-text"
                          : "link-underline text-small text-ink-soft transition-colors duration-200 hover:text-ink"
                      }
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          ))}
        </div>

        <div
          className={
            onStage
              ? "mt-12 flex flex-col gap-2 border-t border-stage-line pt-6 text-micro text-stage-muted sm:flex-row sm:items-center sm:justify-between"
              : "mt-12 flex flex-col gap-2 border-t border-hairline pt-6 text-micro text-ink-muted sm:flex-row sm:items-center sm:justify-between"
          }
        >
          <p>Strangerly is an independent project. Adults only.</p>
          <p>Reports are read by a person, not in real time. Not a crisis service.</p>
        </div>
      </div>
    </footer>
  );
}
