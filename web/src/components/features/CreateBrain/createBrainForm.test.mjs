import assert from "node:assert/strict";

import { createBrainFailure, sourceIDCandidate } from "./createBrainForm.ts";

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
