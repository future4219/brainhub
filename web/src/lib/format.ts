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

export type ConnectionStep = { number: string; title: string; body: string; code?: string; hint?: string };

export function connectionSteps(mcpURL: string, clientID: string | null): ConnectionStep[] {
  return [
    { number: "01", title: "設定 → コネクタ → カスタムコネクタを追加", body: "claude.ai の設定から進む。デスクトップアプリでも同じ場所にある。" },
    { number: "02", title: "MCP URL を貼って保存する。ここで失敗する", body: "自動登録が試みられ、OAuth Client ID がないと言われて止まる。次の手順で埋める。", code: mcpURL, hint: "→ 「コネクタを編集して OAuth クライアント ID を追加してください」" },
    { number: "03", title: "作られたコネクタを編集し、Client ID を貼る", body: clientID ? "失敗したコネクタの OAuth Client ID 欄に貼って保存する。シークレットは不要。" : "先に右の「発行」から Client ID を作り、OAuth Client ID 欄に貼る。", code: clientID ?? undefined },
    { number: "04", title: "接続 → 認可を許可する", body: "brainhub の認可画面に飛ぶ。許可すると、現在見られるすべての脳への読み取り権限が付く。" },
    { number: "05", title: "会話から呼べることを確かめる", body: "見られる脳を2つ指定し、それぞれから情報を引かせる。" },
  ];
}
