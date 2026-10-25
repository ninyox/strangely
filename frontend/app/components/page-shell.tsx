import { SiteFooter } from "./site-footer";
import { SiteHeader } from "./site-header";

/**
 * The shell for the written pages.
 *
 * An editorial layout: a kicker and a serif title, a sticky table of contents on
 * a wide screen, and body copy held to a 66-character measure with hairline
 * rules between sections. Nothing here is a card, because prose is not a card.
 */
export function PageShell({
  kicker,
  title,
  intro,
  updated,
  sections,
  children,
}: {
  kicker: string;
  title: string;
  intro?: string;
  updated?: string;
  /** Anchor ids, in order, for the sticky contents list. */
  sections?: { id: string; label: string }[];
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen flex-col">
      <SiteHeader />

      <main className="flex-1">
        <header className="border-b border-hairline">
          <div className="shell py-14 lg:py-20">
            <p className="mono-label text-ink-muted">{kicker}</p>
            <h1 className="mt-5 max-w-3xl font-display text-display-lg text-ink">{title}</h1>
            {intro ? (
              <p className="measure mt-6 text-lead text-ink-soft">{intro}</p>
            ) : null}
            {updated ? <p className="mono-label mt-6 text-ink-muted">{updated}</p> : null}
          </div>
        </header>

        <div className="shell py-12 lg:py-16">
          <div className="grid gap-12 lg:grid-cols-[13rem_1fr] lg:gap-20">
            {sections?.length ? (
              <nav aria-label="On this page" className="lg:sticky lg:top-28 lg:self-start">
                <p className="mono-label text-ink-muted">On this page</p>
                <ul className="mt-4 flex flex-col gap-2.5 border-l border-hairline pl-4">
                  {sections.map((section) => (
                    <li key={section.id}>
                      <a
                        href={`#${section.id}`}
                        className="link-underline text-small text-ink-soft transition-colors duration-200 hover:text-ink"
                      >
                        {section.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>
            ) : (
              <div aria-hidden="true" />
            )}

            <div className="flex min-w-0 flex-col gap-12">{children}</div>
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  );
}

/** One titled section of copy. The id doubles as the anchor for the contents. */
export function PageSection({
  id,
  title,
  children,
}: {
  id: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section id={id} className="scroll-mt-28">
      <h2 className="font-display text-display-sm text-ink">{title}</h2>
      <div className="measure mt-4 flex flex-col gap-4 text-body text-ink-soft">{children}</div>
    </section>
  );
}

/** A bulleted list with the accent used as the bullet. */
export function PageList({ items }: { items: React.ReactNode[] }) {
  return (
    <ul className="flex flex-col gap-3">
      {items.map((item, index) => (
        <li key={index} className="relative pl-6">
          <span
            aria-hidden="true"
            className="absolute left-0 top-[0.6em] size-1.5 rounded-full bg-signal"
          />
          {item}
        </li>
      ))}
    </ul>
  );
}
