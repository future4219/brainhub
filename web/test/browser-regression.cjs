const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");
const assert = require("node:assert/strict");
const baseURL = process.env.BRAINHUB_TEST_URL || "http://localhost:3000";
(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1280, height: 960 },
  });
  const page = await context.newPage();
  page.setDefaultTimeout(10000);
  try {
    let tokens = [],
      invitations = [],
      claudePrepared = false;
    let loggedIn = true,
      prepared = false,
      empty = false,
      fail = false,
      decision = "",
      requestedScope = "read write",
      grantedScope = "";
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    const visible = [
      {
        source_id: "work",
        name: "仕事の脳",
        state: "ready",
        role: "owner",
        can_write: true,
      },
      {
        source_id: "life",
        name: "暮らしの脳",
        state: "ready",
        role: "reader",
        can_write: false,
      },
      {
        source_id: "reference-library",
        name: "Reference Library — Sample Collection",
        state: "ready",
        role: "reader",
        can_write: false,
      },
      {
        source_id: "personal-notes",
        name: "暮らしの記録",
        state: "ready",
        role: "owner",
        can_write: true,
      },
    ];
    const brain = {
      id: "b1",
      source_id: "work",
      name: "仕事の脳",
      state: "ready",
      owner_id: "owner",
      visibility: "private",
      description: "",
      created_at: "2026-09-06",
      updated_at: "2026-09-06",
    };
    await page.route("**/api/**", async (route) => {
      const path = new URL(route.request().url()).pathname;
      let status = 200,
        body;
      if (path === "/api/me") {
        status = loggedIn ? 200 : 401;
        body = loggedIn
          ? { id: "owner", name: "Test user", email: "test@example.test" }
          : {};
      } else if (path === "/api/auth/login") {
        loggedIn = true;
        body = { id: "owner", name: "Test user", email: "test@example.test" };
      } else if (path === "/api/config")
        body = { mcp_url: "https://brainhub.example/mcp", web_url: baseURL };
      else if (path === "/api/brains") body = [brain];
      else if (path === "/api/brains/work") body = brain;
      else if (path === "/api/brains/work/pages") body = [];
      else if (path === "/api/brains/work/pages/notes/example")
        body = {
          slug: "notes/example",
          title: "Example page",
          type: "note",
          compiled_truth: "Readable Markdown body",
          timeline: "",
          tags: [],
          superseded_by: null,
          updated_at: "2026-09-07",
        };
      else if (path.endsWith("/page-types"))
        throw new Error("Retired page-types API called");
      else if (path === "/api/mcp/connection") {
        status = fail ? 500 : 200;
        body = {
          client: claudePrepared
            ? { id: "claude-client", name: "claude-web" }
            : null,
          codex_client: prepared ? { id: "codex-client", name: "codex" } : null,
          visible_brains: empty ? [] : visible,
          cli_tokens: tokens,
          reader: null,
        };
      } else if (path === "/api/mcp/clients/codex") {
        prepared = true;
        status = 201;
        body = { id: "codex-client", name: "codex" };
      } else if (path === "/api/mcp/clients/claude-web") {
        claudePrepared = true;
        status = 201;
        body = { id: "claude-client", name: "claude-web" };
      } else if (path === "/api/mcp/tokens") {
        const token = {
          id: `token-${tokens.length + 1}`,
          label: JSON.parse(route.request().postData()).label,
          created_at: "2026-09-07",
          expires_at: "2099-12-31",
        };
        tokens.push(token);
        status = 201;
        body = { ...token, token: "synthetic-secret-" + token.id };
      } else if (path.startsWith("/api/mcp/tokens/")) {
        tokens = tokens.filter((t) => !path.endsWith("/" + t.id));
        status = 204;
      } else if (path === "/api/brains/work/invitations") {
        if (route.request().method() === "POST") {
          const invitation = {
            ...JSON.parse(route.request().postData()),
            id: "invite-1",
            state: "pending",
            created_at: "2026-09-07",
            accepted_at: null,
            accepted_by: null,
          };
          invitations.push(invitation);
          status = 201;
          body = { ...invitation, token: "synthetic-invite" };
        } else body = invitations;
      } else if (path === "/api/oauth/authorization") {
        if (route.request().method() === "POST") {
          decision = new URLSearchParams(route.request().postData()).get(
            "decision",
          );
          grantedScope = new URLSearchParams(route.request().postData()).get(
            "granted_scope",
          );
          body = { redirect_uri: baseURL + "/settings/connections" };
        } else
          body = {
            client_id: "codex-client",
            client_name: "codex",
            scope: requestedScope,
            visible_brains: visible,
          };
      } else throw new Error("Unexpected API request: " + path);
      await route.fulfill({
        status,
        contentType: "application/json",
        body: body === undefined ? "" : JSON.stringify(body),
      });
    });
    await page.addInitScript(() => {
      window.clipboardDenied = false;
      window.copiedValues = [];
      Object.defineProperty(navigator, "clipboard", {
        value: {
          writeText: async (value) => {
            if (window.clipboardDenied) throw new Error("clipboard denied");
            window.copiedValues.push(value);
          },
        },
      });
    });
    await page.goto(baseURL + "/settings/connections");
    await page
      .getByRole("button", { name: "Codexの接続を準備", exact: true })
      .click();
    await page
      .getByText(
        "codex mcp add brainhub --url 'https://brainhub.example/mcp' --oauth-client-id 'codex-client'",
        { exact: true },
      )
      .waitFor();
    assert.equal(await page.getByText("仕事の脳", { exact: true }).count(), 1);
    // The shared clipboard reports success and failure without losing other keys.
    await page
      .getByRole("button", { name: "コピー", exact: true })
      .first()
      .click();
    await page
      .getByRole("button", { name: "コピー済み", exact: true })
      .waitFor();
    assert.match(
      await page.evaluate(() => window.copiedValues.at(-1)),
      /^codex mcp add brainhub/,
    );
    await page.evaluate(() => {
      window.clipboardDenied = true;
    });
    await page.getByRole("button", { name: "コピー済み", exact: true }).click();
    await page
      .getByText("コピーできません。コマンドを選択してコピーしてください。", {
        exact: true,
      })
      .waitFor();
    await page.evaluate(() => {
      window.clipboardDenied = false;
    });

    // The extracted manual token panel still issues, copies, resets, and revokes.
    await page.getByText("手動トークンで接続する", { exact: true }).click();
    await page.getByPlaceholder("codex", { exact: true }).fill("qa-token");
    await page
      .getByRole("button", { name: "トークンを発行", exact: true })
      .click();
    await page.getByText("synthetic-secret-token-1", { exact: true }).waitFor();
    const tokenCopy = () =>
      page
        .getByText("CLI TOKEN — 今だけ表示", { exact: true })
        .locator("..")
        .getByRole("button");
    await tokenCopy().click();
    assert.equal(await tokenCopy().textContent(), "コピー済み");
    await page.getByPlaceholder("codex", { exact: true }).fill("qa-token-two");
    await page
      .getByRole("button", { name: "トークンを発行", exact: true })
      .click();
    await page.getByText("synthetic-secret-token-2", { exact: true }).waitFor();
    assert.equal(await tokenCopy().textContent(), "コピー");
    await page
      .getByRole("button", { name: "失効", exact: true })
      .last()
      .click();
    await page
      .getByText("synthetic-secret-token-2", { exact: true })
      .waitFor({ state: "detached" });
    assert.equal(tokens.length, 1);

    await page.getByText("Claude Webに接続する", { exact: true }).click();
    await page
      .getByRole("button", { name: "Claude Webの接続を準備", exact: true })
      .click();
    await page.getByText("claude-client", { exact: true }).first().waitFor();

    // Invitation copying uses the same hook, but keeps its own state.
    await page.goto(baseURL + "/brains/work?tab=invites");
    await page.locator('input[name="expires_at"]').fill("2099-12-31T12:00");
    await page.getByRole("button", { name: "招待を作る", exact: true }).click();
    await page
      .getByText(baseURL + "/invite/synthetic-invite", { exact: true })
      .waitFor();
    const inviteCopy = () =>
      page
        .getByText("招待リンク", { exact: true })
        .locator("..")
        .getByRole("button");
    await page.evaluate(() => {
      window.clipboardDenied = true;
    });
    await inviteCopy().click();
    await page
      .getByText("コピーできません。リンクを選択してコピーしてください。", {
        exact: true,
      })
      .waitFor();
    await page.evaluate(() => {
      window.clipboardDenied = false;
    });
    await inviteCopy().click();
    assert.equal(await inviteCopy().textContent(), "コピー済み");
    assert.equal(
      await page.evaluate(() => window.copiedValues.at(-1)),
      baseURL + "/invite/synthetic-invite",
    );

    await page.goto(baseURL + "/brains/work?tab=connect");
    await page.waitForURL("**/settings/connections");
    await page.goto(baseURL + "/brains/work");
    await page.getByRole("navigation", { name: "脳の画面" }).waitFor();
    assert.equal(
      await page
        .getByRole("navigation", { name: "脳の画面" })
        .getByText("接続", { exact: true })
        .count(),
      0,
    );
    assert.equal(
      await page
        .getByRole("link", { name: "ページを作成", exact: true })
        .count(),
      0,
    );
    await page.goto(baseURL + "/brains/work/pages/notes/example");
    await page.getByText("Readable Markdown body", { exact: true }).waitFor();
    assert.equal(
      await page.getByRole("link", { name: "編集", exact: true }).count(),
      0,
    );
    for (const retired of [
      "/brains/work/page-editor/new",
      "/brains/work/page-editor/edit/notes/example",
    ]) {
      await page.goto(baseURL + retired);
      await page
        .getByRole("heading", { name: "ページが見つかりません", exact: true })
        .waitFor();
    }
    await page.goto(
      baseURL + "/oauth/authorize?client_id=codex-client&state=test",
    );
    await page.getByText("現在読める脳", { exact: true }).waitFor();
    assert.equal(await page.getByText("仕事の脳", { exact: true }).count(), 1);
    assert.equal(
      await page.getByText("暮らしの脳", { exact: true }).count(),
      1,
    );
    assert.equal(
      await page.getByRole("radio", { name: /読み取り・書き込み/ }).isChecked(),
      true,
    );
    assert.equal(await page.getByText("読み書き", { exact: true }).count(), 2);
    await page.getByRole("radio", { name: /読み取りのみ/ }).check();
    assert.equal(await page.getByText("読み書き", { exact: true }).count(), 0);
    await page.getByRole("radio", { name: /読み取り・書き込み/ }).check();
    const buttonColors = await page
      .getByRole("button", { name: "許可する", exact: true })
      .evaluate((el) => ({
        text: getComputedStyle(el).color,
        bg: getComputedStyle(el).backgroundColor,
      }));
    assert.notEqual(buttonColors.text, buttonColors.bg);
    await page.setViewportSize({ width: 1920, height: 1080 });
    const card = await page.getByRole("main").boundingBox();
    assert.ok(card.width >= 560);
    await page.setViewportSize({ width: 390, height: 844 });
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    );
    await page.setViewportSize({ width: 1280, height: 960 });
    await page.getByRole("button", { name: "許可する", exact: true }).click();
    await page.waitForURL("**/settings/connections");
    assert.equal(decision, "approve");
    assert.equal(grantedScope, "read write");
    await page.goto(
      baseURL + "/oauth/authorize?client_id=codex-client&state=read-only",
    );
    await page.getByRole("radio", { name: /読み取りのみ/ }).check();
    await page.getByRole("button", { name: "許可する", exact: true }).click();
    await page.waitForURL("**/settings/connections");
    assert.equal(grantedScope, "read");
    requestedScope = "read";
    await page.goto(
      baseURL + "/oauth/authorize?client_id=codex-client&state=explicit-read",
    );
    await page.getByText("現在読める脳", { exact: true }).waitFor();
    assert.equal(
      await page
        .getByRole("radio", { name: /読み取り・書き込み/ })
        .isDisabled(),
      true,
    );
    assert.equal(
      await page.getByRole("radio", { name: /読み取りのみ/ }).isChecked(),
      true,
    );
    await page.getByRole("button", { name: "許可しない", exact: true }).click();
    await page.waitForURL("**/settings/connections");
    assert.equal(decision, "deny");
    requestedScope = "read write";
    await page.setViewportSize({ width: 390, height: 844 });
    await page.reload();
    await page.getByText("CodexにBrainhubを追加", { exact: true }).waitFor();
    assert.ok(
      await page
        .getByRole("link", { name: "AIとの接続", exact: true })
        .last()
        .isVisible(),
    );
    assert.ok(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    );
    empty = true;
    await page.reload();
    await page
      .getByText(
        "現在読める脳はありません。脳を作るか、招待を受けると利用できます。",
        { exact: true },
      )
      .waitFor();
    fail = true;
    await page.reload();
    await page
      .getByText("接続情報を読み込めません。再読み込みしてください。", {
        exact: true,
      })
      .waitFor();
    fail = false;
    loggedIn = false;
    await page.reload();
    await page
      .getByRole("main")
      .getByRole("link", { name: "ログイン", exact: true })
      .waitFor();
    assert.equal(
      await page
        .getByRole("button", { name: "Codexの接続を準備", exact: true })
        .count(),
      0,
    );
    const oauthPath =
      "/oauth/authorize?client_id=codex-client&state=preserve%2Bstate&code_challenge=challenge&redirect_uri=http%3A%2F%2F127.0.0.1%3A41197%2Fcallback";
    const loginPath = "/login?next=" + encodeURIComponent(oauthPath);
    await page.goto(baseURL + loginPath);
    await page
      .getByRole("textbox", { name: "メールアドレス", exact: true })
      .fill("test@example.test");
    await page.locator('input[name="password"]').fill("test-password-long");
    await page.getByRole("button", { name: "ログイン", exact: true }).click();
    await page.waitForURL(baseURL + oauthPath);
    await page.getByText("現在読める脳", { exact: true }).waitFor();
    assert.equal(
      new URL(page.url()).searchParams.get("state"),
      "preserve+state",
    );
    await page.goto(baseURL + loginPath);
    await page.waitForURL(baseURL + oauthPath);
    await page.getByText("現在読める脳", { exact: true }).waitFor();
    console.log(
      "PASS: fresh login and already-logged-in user both return to consent with OAuth parameters preserved.",
    );
    assert.deepEqual(errors, []);
    console.log(
      "PASS: Codex setup, legacy redirect, per-brain navigation, scope consent/approve, mobile, empty/error/logged-out states; no browser errors.",
    );
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
