import assert from "node:assert/strict";

import {
  brainTab,
  brainAddressPrefix,
  brainUrl,
  invitationUrl,
  editPageUrl,
  markdownPageHref,
  newPageUrl,
  pageUrl,
} from "./url.ts";
import {
  createBrainFailure,
  sourceIDCandidate,
} from "../components/features/CreateBrain/createBrainForm.ts";
import { connectionSteps, formatDate, pageSignature } from "../lib/format.ts";
import { safeNextPath } from "../lib/security.ts";

assert.equal(brainTab("?tab=invites"), "invites");
assert.equal(brainAddressPrefix("https://brainhub.example.com"), "brainhub.example.com/brains/");
assert.equal(brainAddressPrefix("http://localhost:3000/app/"), "localhost:3000/app/brains/");
assert.equal(sourceIDCandidate(" Product  Research "), "product-research");
assert.equal(sourceIDCandidate("製品 調査"), "");
assert.equal(sourceIDCandidate("A___B -- C"), "ab-c");
assert.ok(sourceIDCandidate("a".repeat(40)).length <= 32);
assert.deepEqual(
  createBrainFailure({
    status: 502,
    message: "brain provisioning failed: shim unavailable",
    data: { brain: { state: "failed" } },
  }),
  {
    messages: [
      "GBrain側で脳を作成できませんでした。",
      "詳細: brain provisioning failed: shim unavailable",
      "この脳は一覧に [failed] として残っています。一覧で状態理由を確認し、GBrainの接続を復旧してから管理者に再処理を依頼してください。",
    ],
    showBrainList: true,
  },
);
assert.equal(brainTab("?connect=1"), "connect");
assert.equal(brainUrl("brainhub", "connect"), "/brains/brainhub?tab=connect");
assert.equal(invitationUrl("a/b"), "/invite/a%2Fb");
assert.equal(pageUrl("brainhub", "decisions/a b"), "/brains/brainhub/pages/decisions/a%20b");
assert.equal(markdownPageHref("eval", "people/tomoko-sato"), "/brains/eval/pages/people/tomoko-sato");
assert.equal(markdownPageHref("eval", "people/tomoko-sato#bio"), "/brains/eval/pages/people/tomoko-sato#bio");
assert.equal(markdownPageHref("eval", "https://example.test/person"), "https://example.test/person");
assert.equal(markdownPageHref("eval", "#orange-mode"), "#orange-mode");
assert.equal(newPageUrl("brainhub"), "/brains/brainhub/page-editor/new");
assert.equal(editPageUrl("brainhub", "decisions/a b"), "/brains/brainhub/page-editor/edit/decisions/a%20b");
assert.equal(safeNextPath("?next=%2Finvite%2Fabc", "/"), "/invite/abc");
assert.equal(safeNextPath("?next=https%3A%2F%2Fevil.test", "/"), "/");
assert.equal(pageSignature(["idea", "decision", "idea", "note", "research"]), "idea 2 · decision 1 · note 1 · +1");
assert.equal(formatDate("2026-08-14T15:23:56.326Z"), "2026-08-14");
assert.equal(connectionSteps("claude", "https://example.test/mcp", "client-id", "brainhub").length, 5);
assert.match(connectionSteps("codex", "https://example.test/mcp", null, "brainhub")[0].code, /https:\/\/example\.test\/mcp/);
