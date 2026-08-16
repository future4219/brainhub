import * as React from "react";

import { cn } from "@/lib/utils";

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "min-h-24 w-full resize-y rounded-[1px] border border-rule bg-paper px-3 py-2 text-base leading-6 text-ink outline-none placeholder:text-muted disabled:cursor-not-allowed disabled:opacity-50 focus-visible:border-signal focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-signal md:text-sm",
        className,
      )}
      {...props}
    />
  );
}

export { Textarea };
