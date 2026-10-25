import { cn } from "../../lib/utils";

/**
 * The Skip countdown, drawn as a ring.
 *
 * `progress` runs from 1 (the full five second window) down to 0 (Skip is
 * available). It is driven by the server's match timestamp, so what the ring
 * shows and what the backend enforces are the same number.
 */
export function CountdownRing({
  progress,
  size = 22,
  thickness = 2,
  className,
  tone = "ink",
}: {
  progress: number;
  size?: number;
  thickness?: number;
  className?: string;
  tone?: "ink" | "stage" | "available";
}) {
  const clamped = Math.min(1, Math.max(0, progress));
  const radius = (size - thickness) / 2;
  const circumference = 2 * Math.PI * radius;
  const dash = circumference * clamped;

  const track =
    tone === "stage" ? "rgba(246,243,239,0.18)" : tone === "available" ? "rgba(255,255,255,0.35)" : "rgba(19,18,17,0.14)";
  const arc =
    tone === "stage" ? "#F04438" : tone === "available" ? "#FFFFFF" : "#F04438";

  return (
    <svg
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className={cn("shrink-0 -rotate-90", className)}
      aria-hidden="true"
    >
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke={track}
        strokeWidth={thickness}
      />
      <circle
        cx={size / 2}
        cy={size / 2}
        r={radius}
        fill="none"
        stroke={arc}
        strokeWidth={thickness}
        strokeLinecap="round"
        strokeDasharray={`${dash} ${circumference}`}
        style={{ transition: "stroke-dasharray 240ms linear" }}
      />
    </svg>
  );
}
