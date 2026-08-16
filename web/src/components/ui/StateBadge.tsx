export function StateBadge({ state }: { state: string }) {
  return (
    <span className="state-badge" data-state={state}>
      [{state}]
    </span>
  );
}
