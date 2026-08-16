import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

export function FormField({
  label,
  hint,
  className,
  children,
}: {
  label: string;
  hint?: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <label className={cn("grid gap-2 border-b border-divider py-4", className)}>
      <span className="font-mono text-xs uppercase tracking-label text-text-muted">
        {label}
      </span>
      {children}
      {hint && <small className="text-xs text-text-muted">{hint}</small>}
    </label>
  );
}
