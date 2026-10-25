import { cn } from "../../../lib/utils";

/**
 * A surface: the one place a shadow is allowed, because a surface is something
 * lifted off the paper. Used by the live panel, the call state panels, and the
 * dialogs. Structure inside a page is carried by hairline rules instead.
 */
export function Surface({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "rounded-surface border border-hairline bg-surface shadow-soft",
        className,
      )}
      {...props}
    />
  );
}

export function SurfaceBody({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("p-6 sm:p-7", className)} {...props} />;
}

/** A quiet, all-caps machine label. */
export function Kicker({
  children,
  className,
  tone = "muted",
}: {
  children: React.ReactNode;
  className?: string;
  tone?: "muted" | "signal" | "stage";
}) {
  return (
    <p
      className={cn(
        "mono-label",
        tone === "muted" && "text-ink-muted",
        tone === "signal" && "text-signal",
        tone === "stage" && "text-stage-muted",
        className,
      )}
    >
      {children}
    </p>
  );
}

/** A short hairline rule used to separate content inside a surface. */
export function Rule({ className }: { className?: string }) {
  return <div className={cn("h-px w-full bg-hairline", className)} aria-hidden="true" />;
}
