import assert from "node:assert/strict";

import {
  brainTab,
  brainAddressPrefix,
  brainUrl,
  invitationUrl,
  markdownPageHref,
  pageUrl,
} from "./url.ts";

assert.equal(brainTab("?tab=invites"), "invites");
assert.equal(
  brainAddressPrefix("https://brainhub.example.com"),
  "brainhub.example.com/brains/",
);
assert.equal(
  brainAddressPrefix("http://localhost:3000/app/"),
  "localhost:3000/app/brains/",
);
assert.equal(brainTab("?connect=1"), "connect");
assert.equal(brainUrl("brainhub", "connect"), "/settings/connections");
assert.equal(invitationUrl("a/b"), "/invite/a%2Fb");
assert.equal(
  pageUrl("brainhub", "decisions/a b"),
  "/brains/brainhub/pages/decisions/a%20b",
);
assert.equal(
  markdownPageHref("eval", "people/tomoko-sato"),
  "/brains/eval/pages/people/tomoko-sato",
);
assert.equal(
  markdownPageHref("eval", "people/tomoko-sato#bio"),
  "/brains/eval/pages/people/tomoko-sato#bio",
);
assert.equal(
  markdownPageHref("eval", "https://example.test/person"),
  "https://example.test/person",
);
assert.equal(markdownPageHref("eval", "#orange-mode"), "#orange-mode");
assert.equal(brainTab("?tab=settings"), "settings");
assert.equal(brainUrl("brainhub", "settings"), "/brains/brainhub?tab=settings");
