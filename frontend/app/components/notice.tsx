import { AlertTriangle, Check, Info } from "lucide-react";

import { cn } from "../../lib/utils";

type NoticeTone = "info" | "warning" | "error" | "success";

const TONES: Record<NoticeTone, { shell: string; icon: typeof Info }> = {
  info: { shell: "border-hairline bg-clay text-ink-soft", icon: Info },
  warning: { shell: "border-warning/25 bg-warning-tint text-warning", icon: AlertTriangle },
  error: { shell: "border-danger/25 bg-danger-tint text-danger", icon: AlertTriangle },
  success: { shell: "border-success/25 bg-success-tint text-success", icon: Check },
};

/**
 * Notice is the single explanation block: a permission problem, a refusal from
 * the server, a safety reminder. It announces itself politely (role="status")
 * unless it is an error, which interrupts (role="alert").
 */
export function Notice({
  tone = "info",
  title,
  children,
  className,
  role,
}: {
  tone?: NoticeTone;
  title?: string;
  children?: React.ReactNode;
  className?: string;
  role?: "status" | "alert";
}) {
  const { shell, icon: Icon } = TONES[tone];

  return (
    <div
      role={role ?? (tone === "error" ? "alert" : "status")}
      className={cn(
        "flex gap-3 rounded-control border px-4 py-3.5 text-small leading-relaxed",
        shell,
        className,
      )}
    >
      <Icon className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <div className="min-w-0">
        {title ? <p className="font-medium">{title}</p> : null}
        {children ? <div className={cn(title && "mt-1", "opacity-95")}>{children}</div> : null}
      </div>
    </div>
  );
}
