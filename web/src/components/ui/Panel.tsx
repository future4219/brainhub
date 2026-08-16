import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function Panel({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "overflow-hidden rounded-panel border border-border bg-sidebar",
        className,
      )}
    >
      {children}
    </div>
  );
}
