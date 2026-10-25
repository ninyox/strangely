import * as React from "react";

import { cn } from "../../../lib/utils";

/**
 * A text field.
 *
 * 44px tall on touch, warm hairline border, and a visible focus treatment that
 * matches every other control in the product. Invalid state is expressed with
 * colour *and* an aria-invalid attribute, never colour alone.
 */
const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<"input">>(
  ({ className, type, ...props }, ref) => (
    <input
      type={type}
      ref={ref}
      className={cn(
        "h-12 w-full rounded-control border border-hairline-strong bg-surface px-4 text-body text-ink",
        "transition-[border-color,box-shadow,background-color] duration-200 ease-settle",
        "placeholder:text-ink-muted",
        "hover:border-ink-muted",
        "focus:border-signal focus:outline-none",
        "disabled:cursor-not-allowed disabled:bg-clay disabled:text-ink-muted",
        "aria-[invalid=true]:border-signal aria-[invalid=true]:bg-signal-tint",
        className,
      )}
      {...props}
    />
  ),
);
Input.displayName = "Input";

export { Input };
