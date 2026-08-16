import { Button } from "@/components/ui/Button";

export function CopyValue({
  label,
  value,
  note,
  copyLabel,
  onCopy,
}: {
  label: string;
  value: string;
  note?: string;
  copyLabel: string;
  onCopy: () => void;
}) {
  return (
    <div className="border-b border-divider py-4 last:border-b-0">
      <div className="mb-2 flex items-center justify-between gap-4">
        <span className="font-mono text-xs text-text-tertiary">{label}</span>
        <Button variant="outline" size="sm" type="button" onClick={onCopy}>
          {copyLabel}
        </Button>
      </div>
      <code className="block break-all rounded-control border border-border-control bg-canvas px-3 py-2 text-sm leading-copy text-text-code">
        {value}
      </code>
      {note && <p className="mt-2 font-mono text-xs text-text-muted">{note}</p>}
    </div>
  );
}
