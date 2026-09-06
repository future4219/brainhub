import assert from "node:assert/strict";

import { safeNextPath } from "./security.ts";

assert.equal(safeNextPath("?next=%2Finvite%2Fabc", "/"), "/invite/abc");
assert.equal(safeNextPath("?next=https%3A%2F%2Fevil.test", "/"), "/");
const consentNext =
  "/oauth/authorize?client_id=codex&state=a%2Bb&code_challenge=abc&redirect_uri=http%3A%2F%2F127.0.0.1%3A41197%2Fcallback%2Ftest";
for (const next of [
  consentNext,
  "/settings/connections",
  "/brains/new",
  "/invite/a%2Fb",
]) {
  assert.equal(safeNextPath(`?next=${encodeURIComponent(next)}`, "/"), next);
}
for (const next of [
  "https://evil.test",
  "//evil.test",
  "/\\evil.test",
  "javascript:alert(1)",
  " /oauth/authorize",
  "/\n/evil.test",
  "/\t/evil.test",
  "/oauth/authorize\u007f",
]) {
  assert.equal(safeNextPath(`?next=${encodeURIComponent(next)}`, "/"), "/");
}
