export function formatDate(value: string) {
  return new Date(value).toISOString().slice(0, 10);
}

export function pageSignature(types: string[]) {
  const counts = new Map<string, number>();
  for (const type of types) counts.set(type, (counts.get(type) ?? 0) + 1);
  const ranked = [...counts].sort(([typeA, countA], [typeB, countB]) => countB - countA || typeA.localeCompare(typeB));
  const leading = ranked.slice(0, 3).map(([type, count]) => `${type} ${count}`);
  if (ranked.length > 3) leading.push(`+${ranked.length - 3}`);
  return leading.join(" · ") || "—";
}

export type ConnectClient = "claude" | "chatgpt" | "codex";
export type ConnectionStep = { number: string; title: string; body: string; code?: string; hint?: string };

export function connectionSteps(client: ConnectClient, mcpURL: string, clientID: string | null, sourceID: string): ConnectionStep[] {
  if (client === "codex") return [
    { number: "01", title: "設定ファイルに MCP サーバーを追記する", body: "~/.codex/config.toml の末尾に追記する。", code: `[mcp_servers.brainhub]\nurl = "${mcpURL}"${clientID ? `\nclient_id = "${clientID}"` : ""}` },
    { number: "02", title: "Codex を再起動する", body: "起動時に設定を読む。常駐している場合は落としてから上げる。" },
    { number: "03", title: "初回はブラウザで認可する", body: "認可 URL を開いて許可する。", code: "codex mcp login brainhub" },
    { number: "04", title: "ツールが見えているか確認する", body: "一覧に brainhub_* が出れば完了。", code: "codex mcp list" },
  ];
  if (client === "chatgpt") return [
    { number: "01", title: "設定 → コネクタ → カスタムコネクタ", body: "ChatGPT 側で MCP サーバーを URL 登録する。ワークスペースによっては管理者の許可が必要。" },
    { number: "02", title: "MCP URL を貼る", body: "右の MCP URL をそのまま使う。", code: mcpURL },
    { number: "03", title: "認証方式に OAuth を選ぶ", body: clientID ? "Client ID を貼る。シークレットは空のまま。" : "先に右の「発行」から Client ID を作る。", code: clientID ?? undefined },
    { number: "04", title: "接続して認可する", body: "認可後、ツールとして呼び出せるようになる。" },
  ];
  return [
    { number: "01", title: "設定 → コネクタ → カスタムコネクタを追加", body: "claude.ai の設定から進む。デスクトップアプリでも同じ場所にある。" },
    { number: "02", title: "MCP URL を貼って保存する。ここで失敗する", body: "自動登録が試みられ、OAuth Client ID がないと言われて止まる。次の手順で埋める。", code: mcpURL, hint: "→ 「コネクタを編集して OAuth クライアント ID を追加してください」" },
    { number: "03", title: "作られたコネクタを編集し、Client ID を貼る", body: clientID ? "失敗したコネクタの OAuth Client ID 欄に貼って保存する。シークレットは不要。" : "先に右の「発行」から Client ID を作り、OAuth Client ID 欄に貼る。", code: clientID ?? undefined },
    { number: "04", title: "接続 → 認可を許可する", body: "brainhub の認可画面に飛ぶ。許可すると読み取り権限が付与される。" },
    { number: "05", title: "会話から呼べることを確かめる", body: "脳の名前を出して呼ばせるのが早い。", code: `${sourceID} の decision から、接続の設計判断を挙げて` },
  ];
}
