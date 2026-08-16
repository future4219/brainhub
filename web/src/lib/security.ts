export function safeNextPath(search: string, fallback: string): string {
  const next = new URLSearchParams(search).get("next");
  return next && /^\/invite\/[^/?#]+$/.test(next) ? next : fallback;
}
