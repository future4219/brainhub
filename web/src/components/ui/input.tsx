import * as React from "react";

import { cn } from "@/lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "h-10 w-full min-w-0 rounded-[1px] border border-rule bg-paper px-3 py-1 text-base text-ink outline-none placeholder:text-muted disabled:cursor-not-allowed disabled:opacity-50 focus-visible:border-signal focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-signal md:text-sm",
        className,
      )}
      {...props}
    />
  );
}

export { Input };
