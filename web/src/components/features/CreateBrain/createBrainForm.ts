export type CreateBrainFailure = {
  messages: string[];
  showBrainList: boolean;
};

type APIErrorLike = {
  status: number;
  message: string;
  data?: unknown;
};

export function sourceIDCandidate(name: string): string {
  return name
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 32)
    .replace(/-+$/g, "");
}

export function createBrainFailure(error?: APIErrorLike): CreateBrainFailure {
  if (!error) {
    return {
      messages: [
        "Brainhub APIに接続できませんでした。",
        "接続を確認して、もう一度作成してください。",
      ],
      showBrainList: false,
    };
  }
  if (error.status === 401) {
    return {
      messages: [
        "ログイン状態を確認できませんでした。",
        "もう一度ログインしてから作成してください。",
      ],
      showBrainList: false,
    };
  }

  const state = responseBrainState(error.data);
  if (error.status === 409 && state === "failed") {
    return {
      messages: [
        "GBrain側に同じアドレスの脳があるため作成できませんでした。",
        `詳細: ${error.message}`,
        "この脳は一覧に [failed] として残っています。一覧で状態理由を確認し、別のアドレスで作成してください。",
      ],
      showBrainList: true,
    };
  }
  if (error.status === 409) {
    return {
      messages: [
        "そのアドレスはすでに使われています。",
        "アドレスを変更して、もう一度作成してください。",
      ],
      showBrainList: false,
    };
  }
  if (error.status === 502 && state === "degraded") {
    return {
      messages: [
        "脳は作成されましたが、GBrainの書き込み接続を準備できませんでした。",
        `詳細: ${error.message}`,
        "この脳は一覧に [degraded] として残っています。一覧から脳を開き、接続画面で書き込み接続を再発行してください。",
      ],
      showBrainList: true,
    };
  }
  if (error.status === 502) {
    return {
      messages: [
        "GBrain側で脳を作成できませんでした。",
        `詳細: ${error.message}`,
        "この脳は一覧に [failed] として残っています。一覧で状態理由を確認し、GBrainの接続を復旧してから管理者に再処理を依頼してください。",
      ],
      showBrainList: true,
    };
  }
  if (error.status === 400) {
    return {
      messages: [
        "入力内容を確認できませんでした。",
        "名前、アドレス、説明、公開範囲を確認して、もう一度作成してください。",
      ],
      showBrainList: false,
    };
  }
  return {
    messages: [
      `脳を作成できませんでした。詳細: ${error.message}`,
      "時間を置いて、もう一度作成してください。",
    ],
    showBrainList: false,
  };
}

function responseBrainState(data: unknown): string | undefined {
  if (!data || typeof data !== "object") return undefined;
  const brain = (data as { brain?: unknown }).brain;
  if (!brain || typeof brain !== "object") return undefined;
  const state = (brain as { state?: unknown }).state;
  return typeof state === "string" ? state : undefined;
}
