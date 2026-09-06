import assert from "node:assert/strict";

import { codexConnectCommand, connectionSteps } from "./connectionSetup.ts";

assert.equal(
  connectionSteps("https://example.test/mcp", "client-id").length,
  5,
);
assert.match(
  connectionSteps("https://example.test/mcp", null)[1].code,
  /https:\/\/example\.test\/mcp/,
);
assert.equal(
  codexConnectCommand("https://example.test/mcp", "client-id"),
  "codex mcp add brainhub --url 'https://example.test/mcp' --oauth-client-id 'client-id'",
);
