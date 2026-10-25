import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";

import { cn } from "../../../lib/utils";

/**
 * A dialog.
 *
 * On a phone it slides up from the bottom edge (where a thumb already is) and
 * on a larger screen it centres. Focus is trapped, Escape closes, and the title
 * is wired to the content for screen readers.
 */
export const Dialog = DialogPrimitive.Root;
export const DialogTrigger = DialogPrimitive.Trigger;
export const DialogClose = DialogPrimitive.Close;

export function DialogContent({
  className,
  children,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Content>) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-ink/50 backdrop-blur-[3px] data-[state=open]:animate-in" />
      <DialogPrimitive.Content
        className={cn(
          "fixed z-50 flex flex-col overflow-y-auto bg-surface shadow-lift",
          // Phone: a sheet pinned to the bottom edge.
          "inset-x-0 bottom-0 max-h-[92vh] rounded-t-surface border-t border-hairline",
          // Tablet and up: a centred card.
          "sm:inset-x-auto sm:bottom-auto sm:left-1/2 sm:top-1/2 sm:max-h-[88vh] sm:w-[min(34rem,calc(100vw-3rem))] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-surface sm:border",
          "p-6 pb-[max(1.5rem,env(safe-area-inset-bottom))] sm:p-7",
          className,
        )}
        {...props}
      >
        <DialogPrimitive.Close
          aria-label="Close"
          className="absolute right-4 top-4 grid size-9 place-items-center rounded-control text-ink-muted transition-colors duration-200 hover:bg-clay hover:text-ink sm:right-5 sm:top-5"
        >
          <X className="size-4" aria-hidden="true" />
        </DialogPrimitive.Close>
        {children}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

export function DialogTitle({ className, ...props }: React.ComponentProps<typeof DialogPrimitive.Title>) {
  return (
    <DialogPrimitive.Title
      className={cn("font-display text-display-sm text-ink pr-10", className)}
      {...props}
    />
  );
}

export function DialogDescription({
  className,
  ...props
}: React.ComponentProps<typeof DialogPrimitive.Description>) {
  return (
    <DialogPrimitive.Description
      className={cn("mt-2.5 text-small leading-relaxed text-ink-soft", className)}
      {...props}
    />
  );
}

export function DialogFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "mt-7 flex flex-col-reverse gap-2.5 sm:flex-row sm:items-center sm:justify-end",
        className,
      )}
      {...props}
    />
  );
}
