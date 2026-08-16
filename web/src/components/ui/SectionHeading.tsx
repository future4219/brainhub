import type { ReactNode } from "react";

export function SectionHeading({
  eyebrow,
  title,
  meta,
  id,
}: {
  eyebrow: string;
  title: string;
  meta?: ReactNode;
  id?: string;
}) {
  return (
    <header className="flex items-center justify-between gap-6 border-b border-border px-4 py-3">
      <p className="font-mono text-xs uppercase tracking-label text-text-muted">
        {eyebrow}
      </p>
      <h2 id={id} className="sr-only">
        {title}
      </h2>
      {meta}
    </header>
  );
}
