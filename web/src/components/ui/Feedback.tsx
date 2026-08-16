import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function Feedback({
  kind,
  children,
  className,
}: {
  kind: "loading" | "error" | "empty" | "info";
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "border-b border-divider px-4 py-8 text-center text-ui",
        kind === "loading" && "font-mono text-text-muted",
        kind === "error" && "bg-surface text-text",
        kind === "empty" && "text-text-secondary",
        kind === "info" && "text-left text-text-secondary",
        className,
      )}
      role={
        kind === "error" ? "alert" : kind === "loading" ? "status" : undefined
      }
    >
      {children}
    </div>
  );
}
