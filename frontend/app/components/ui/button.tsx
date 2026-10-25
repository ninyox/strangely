import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../../../lib/utils";

/**
 * Buttons.
 *
 * Seven variants, five sizes, one focus treatment (the global :focus-visible
 * outline). The primary button is the only element in the product that lifts on
 * hover, which is what keeps it unmistakably the action.
 */
const buttonVariants = cva(
  [
    "relative inline-flex shrink-0 items-center justify-center gap-2",
    "whitespace-nowrap font-medium",
    "transition-[transform,background-color,color,border-color,box-shadow,opacity]",
    "duration-200 ease-settle",
    "disabled:pointer-events-none",
    "[&_svg]:pointer-events-none [&_svg]:shrink-0",
  ].join(" "),
  {
    variants: {
      variant: {
        primary:
          "bg-signal text-white shadow-soft hover:bg-signal-deep hover:-translate-y-px hover:shadow-lift active:translate-y-0 disabled:bg-clay disabled:text-ink-muted disabled:shadow-none",
        ink: "bg-ink text-white hover:bg-ink-soft disabled:bg-clay disabled:text-ink-muted",
        outline:
          "border border-hairline-strong bg-surface text-ink hover:border-ink-muted hover:bg-clay disabled:border-hairline disabled:bg-transparent disabled:text-ink-muted",
        ghost:
          "text-ink-soft hover:bg-clay hover:text-ink disabled:bg-transparent disabled:text-ink-muted",
        danger: "bg-danger text-white hover:bg-[#8f1c13] disabled:bg-clay disabled:text-ink-muted",
        /* For the dark call stage. */
        stage:
          "border border-white/15 bg-white/8 text-stage-text hover:border-white/30 hover:bg-white/16 disabled:border-stage-line disabled:bg-transparent disabled:text-stage-muted",
        /* A solid light button for the saturated bands, where the accent is
           already the background. */
        inverse:
          "bg-white text-signal shadow-soft hover:-translate-y-px hover:bg-white/92 hover:shadow-lift active:translate-y-0",
        link: "px-0 text-signal underline-offset-4 hover:underline",
      },
      size: {
        sm: "h-9 rounded-control px-3.5 text-small [&_svg]:size-4",
        md: "h-11 rounded-control px-5 text-small [&_svg]:size-4",
        lg: "h-12 rounded-control px-6 text-body [&_svg]:size-[1.125rem]",
        xl: "h-14 rounded-surface px-7 text-body-lg [&_svg]:size-5",
        icon: "size-11 rounded-control [&_svg]:size-[1.125rem]",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />;
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
