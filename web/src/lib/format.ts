export function formatDate(value: string) {
  return new Date(value).toISOString().slice(0, 10);
}

export function pageSignature(types: string[]) {
  const counts = new Map<string, number>();
  for (const type of types) counts.set(type, (counts.get(type) ?? 0) + 1);
  const ranked = [...counts].sort(
    ([typeA, countA], [typeB, countB]) =>
      countB - countA || typeA.localeCompare(typeB),
  );
  const leading = ranked.slice(0, 3).map(([type, count]) => `${type} ${count}`);
  if (ranked.length > 3) leading.push(`+${ranked.length - 3}`);
  return leading.join(" · ") || "—";
}

export function copyLabel(state?: "copied" | "failed") {
  return state === "copied" ? "コピー済み" : "コピー";
}
