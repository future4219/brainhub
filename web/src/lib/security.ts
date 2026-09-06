export function safeNextPath(search: string, fallback: string): string {
  const next = new URLSearchParams(search).get("next");
  // Keep local routes and their query strings, including OAuth state and PKCE.
  return next && next.startsWith("/") && !next.startsWith("//") && !/[\\\u0000-\u0020\u007f]/.test(next)
    ? next
    : fallback;
}
