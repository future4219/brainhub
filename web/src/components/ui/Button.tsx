import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center whitespace-nowrap rounded-control border font-semibold transition-colors outline-none disabled:pointer-events-none disabled:opacity-50 motion-reduce:transition-none",
  {
    variants: {
      variant: {
        default: "border-text bg-text text-canvas hover:bg-text-hover",
        outline: "border-border-interactive bg-surface text-text hover:border-text-muted hover:bg-surface-hover",
        ghost: "border-transparent bg-transparent text-text-secondary hover:text-text-hover",
        danger:
          "border-[var(--bh-color-failed-border)] bg-[var(--bh-color-failed-bg)] text-[var(--bh-color-failed-text)] hover:brightness-110",
      },
      size: {
        default: "h-control px-4 text-ui",
        sm: "h-control-sm px-3 text-xs",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: React.ComponentProps<"button"> & VariantProps<typeof buttonVariants>) {
  return (
    <button
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}

export { Button, buttonVariants };
