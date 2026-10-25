import { cn } from "../../lib/utils";

/**
 * The Strangerly mark.
 *
 * Two rings, one intersection: two people who did not know each other, briefly
 * overlapping. The filled lens is the only place the signal colour appears in
 * the mark, so the identity and the action colour stay the same idea.
 */
export function Mark({ className, tone = "ink" }: { className?: string; tone?: "ink" | "stage" | "mono" }) {
  const stroke = tone === "stage" ? "#F6F3EF" : "#131211";
  const fill = tone === "mono" ? "#FFFFFF" : "#F04438";

  return (
    <svg
      viewBox="0 0 36 36"
      className={cn("size-6", className)}
      role="img"
      aria-label="Strangerly"
      fill="none"
    >
      <circle cx="13" cy="18" r="9" stroke={stroke} strokeWidth="1.75" />
      <circle cx="23" cy="18" r="9" stroke={stroke} strokeWidth="1.75" />
      <path d="M18 10.52 A9 9 0 0 1 18 25.48 A9 9 0 0 1 18 10.52 Z" fill={fill} />
    </svg>
  );
}

/** The mark plus the wordmark, set in the editorial face. */
export function Wordmark({
  className,
  tone = "ink",
  showMark = true,
}: {
  className?: string;
  tone?: "ink" | "stage";
  showMark?: boolean;
}) {
  return (
    <span className={cn("inline-flex items-center gap-2.5", className)}>
      {showMark ? <Mark tone={tone} className="size-[1.375rem]" /> : null}
      <span
        className={cn(
          "font-display text-[1.3125rem] font-medium tracking-[-0.02em]",
          tone === "stage" ? "text-stage-text" : "text-ink",
        )}
      >
        Strangerly
      </span>
    </span>
  );
}
