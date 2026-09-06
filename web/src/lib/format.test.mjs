import assert from "node:assert/strict";

import { formatDate, pageSignature } from "./format.ts";

assert.equal(
  pageSignature(["idea", "decision", "idea", "note", "research"]),
  "idea 2 · decision 1 · note 1 · +1",
);
assert.equal(formatDate("2026-08-14T15:23:56.326Z"), "2026-08-14");
