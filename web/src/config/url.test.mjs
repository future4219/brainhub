import assert from "node:assert/strict";

import {
  brainTab,
  brainUrl,
  invitationUrl,
  editPageUrl,
  newPageUrl,
  pageUrl,
} from "./url.ts";
import { connectionSteps, formatDate, pageSignature } from "../lib/format.ts";
import { safeNextPath } from "../lib/security.ts";

assert.equal(brainTab("?tab=invites"), "invites");
assert.equal(brainTab("?connect=1"), "connect");
assert.equal(brainUrl("brainhub", "connect"), "/brains/brainhub?tab=connect");
assert.equal(invitationUrl("a/b"), "/invite/a%2Fb");
assert.equal(pageUrl("brainhub", "decisions/a b"), "/brains/brainhub/pages/decisions/a%20b");
assert.equal(newPageUrl("brainhub"), "/brains/brainhub/page-editor/new");
assert.equal(editPageUrl("brainhub", "decisions/a b"), "/brains/brainhub/page-editor/edit/decisions/a%20b");
assert.equal(safeNextPath("?next=%2Finvite%2Fabc", "/"), "/invite/abc");
assert.equal(safeNextPath("?next=https%3A%2F%2Fevil.test", "/"), "/");
assert.equal(pageSignature(["idea", "decision", "idea", "note", "research"]), "idea 2 · decision 1 · note 1 · +1");
assert.equal(formatDate("2026-08-14T15:23:56.326Z"), "2026-08-14");
assert.equal(connectionSteps("claude", "https://example.test/mcp", "client-id", "brainhub").length, 5);
assert.match(connectionSteps("codex", "https://example.test/mcp", null, "brainhub")[0].code, /https:\/\/example\.test\/mcp/);
