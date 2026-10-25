import { cn } from "../../lib/utils";

/**
 * Loading indicators. The spinner is a hairline ring that matches the countdown
 * ring used for the Skip lock, so "waiting" looks the same everywhere.
 */
export function Spinner({
  className,
  label,
  tone = "ink",
}: {
  className?: string;
  label?: string;
  tone?: "ink" | "stage";
}) {
  return (
    <span
      className={cn("inline-flex items-center gap-2.5", className)}
      role="status"
      aria-live="polite"
    >
      <span
        aria-hidden="true"
        className={cn(
          "size-4 animate-spin rounded-full border-[1.5px] border-current border-t-transparent",
          tone === "stage" ? "text-stage-muted" : "text-ink-muted",
        )}
      />
      {label ? <span className="text-small">{label}</span> : null}
    </span>
  );
}

/**
 * The signal rings: concentric circles expanding from one point. Used while
 * searching for a partner, and as a quiet background motif behind the hero.
 */
export function SignalRings({
  className,
  tone = "signal",
  animate = true,
  count = 3,
}: {
  className?: string;
  tone?: "signal" | "stage" | "paper";
  animate?: boolean;
  count?: number;
}) {
  const color = tone === "signal" ? "#F04438" : tone === "stage" ? "#F6F3EF" : "#D5CFC6";

  return (
    <span
      aria-hidden="true"
      className={cn("pointer-events-none absolute inset-0 grid place-items-center", className)}
    >
      {Array.from({ length: count }).map((_, index) => (
        <span
          key={index}
          className={cn(
            "absolute size-full rounded-full border",
            animate && "strg-ring",
          )}
          style={{
            borderColor: color,
            opacity: animate ? undefined : 0.14 - index * 0.03,
            animationDelay: animate ? `${index * 1.05}s` : undefined,
          }}
        />
      ))}
    </span>
  );
}

/**
 * The live dot: a filled circle with a soft halo. It appears anywhere the
 * product is showing that something is happening right now.
 */
export function LiveDot({ className, tone = "signal" }: { className?: string; tone?: "signal" | "success" }) {
  const color = tone === "success" ? "#067647" : "#F04438";

  return (
    <span className={cn("relative grid size-2.5 place-items-center", className)} aria-hidden="true">
      <span
        className="strg-live absolute size-full rounded-full"
        style={{ backgroundColor: color, opacity: 0.35 }}
      />
      <span className="relative size-1.5 rounded-full" style={{ backgroundColor: color }} />
    </span>
  );
}
