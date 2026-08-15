import { useEffect, useMemo, useState, type FormEvent } from "react";
import {
  APIError,
  createBrain,
  getBrain,
  getCurrentUser,
  getPublicConfig,
  listBrains,
  listPages,
  login,
  logout,
  register,
  type Brain,
  type Page,
  type PublicConfig,
  type User,
} from "./api";

type Viewer = User | null | undefined;

function App() {
  const [viewer, setViewer] = useState<Viewer>(undefined);
  const [sessionUnavailable, setSessionUnavailable] = useState(false);

  useEffect(() => {
    void getCurrentUser()
      .then(setViewer)
      .catch(() => {
        setViewer(null);
        setSessionUnavailable(true);
      });
  }, []);

  async function handleLogout() {
    try {
      await logout();
      window.location.assign("/");
    } catch {
      window.alert("ログアウトできません。接続を確認してもう一度お試しください。");
    }
  }

  const shared = { viewer, sessionUnavailable, onLogout: handleLogout };
  const path = window.location.pathname;
  if (path === "/" || path === "") return <BrainListPage {...shared} />;
  if (path === "/login") return <AuthPage {...shared} mode="login" />;
  if (path === "/register") return <AuthPage {...shared} mode="register" />;
  if (path === "/brains/new") return <CreateBrainPage {...shared} />;

  const match = path.match(/^\/brains\/([a-z0-9-]+)\/?$/);
  if (match) return <BrainDetailPage {...shared} sourceID={match[1]} />;
  return <StatusPage {...shared} label="404" title="ページが見つかりません" />;
}

type PageProps = {
  viewer: Viewer;
  sessionUnavailable: boolean;
  onLogout: () => Promise<void>;
};

function BrainListPage(props: PageProps) {
  const [brains, setBrains] = useState<Brain[] | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    document.title = "brainhub — brains";
    void listBrains()
      .then(setBrains)
      .catch(() => setError(true));
  }, []);

  return (
    <main className="shell">
      <SiteHeader {...props} />

      <header className="page-intro">
        <div>
          <p className="eyebrow">Knowledge repositories</p>
          <h1>Brains</h1>
        </div>
        <p>中身を確かめ、必要な知識だけを普段使っているAIへ接続する。</p>
      </header>

      <section aria-labelledby="brain-list-heading">
        <div className="list-heading">
          <h2 id="brain-list-heading">{props.viewer ? "Accessible brains" : "Public brains"}</h2>
          {brains && <span className="mono">{brains.length} repositories</span>}
        </div>

        {error && <ErrorPanel message="脳の一覧を取得できません。接続を確認して再読み込みしてください。" />}
        {!error && brains === null && <Loading />}
        {brains?.length === 0 && (
          <Empty
            message={
              props.viewer
                ? "まだ脳がありません。「脳を作る」から最初の領域を追加できます。"
                : "公開中の脳はありません。readyになったpublicの脳がここに表示されます。"
            }
          />
        )}
        {brains && brains.length > 0 && (
          <div className="brain-table">
            <div className="brain-columns" aria-hidden="true">
              <span>brain / source</span>
              <span>visibility</span>
              <span>state</span>
              <span />
            </div>
            {brains.map((brain) => (
              <BrainRow brain={brain} key={brain.id} />
            ))}
          </div>
        )}
      </section>
    </main>
  );
}

function BrainRow({ brain }: { brain: Brain }) {
  const content = (
    <>
      <div className="brain-identity">
        <strong>{brain.name}</strong>
        <span className="mono">{brain.source_id}</span>
        {brain.description && <p>{brain.description}</p>}
      </div>
      <span className="mono brain-visibility">{brain.visibility}</span>
      <span className="mono brain-state">[{brain.state}]</span>
      <span className="row-arrow" aria-hidden="true">
        {brain.state === "ready" ? "→" : "—"}
      </span>
    </>
  );

  if (brain.state !== "ready") {
    return (
      <div className="brain-row unavailable" aria-disabled="true">
        {content}
      </div>
    );
  }
  return (
    <a className="brain-row" href={`/brains/${encodeURIComponent(brain.source_id)}`}>
      {content}
    </a>
  );
}

function BrainDetailPage({ sourceID, ...props }: PageProps & { sourceID: string }) {
  const [brain, setBrain] = useState<Brain | null>(null);
  const [pages, setPages] = useState<Page[] | null>(null);
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [selectedType, setSelectedType] = useState("all");
  const [error, setError] = useState<"not-found" | "load" | null>(null);
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");

  useEffect(() => {
    document.title = `${sourceID} — brainhub`;
    void (async () => {
      try {
        const [loadedBrain, loadedConfig] = await Promise.all([getBrain(sourceID), getPublicConfig()]);
        setBrain(loadedBrain);
        setConfig(loadedConfig);
        if (loadedBrain.state === "ready") setPages(await listPages(sourceID));
      } catch (cause: unknown) {
        setError(cause instanceof APIError && cause.status === 404 ? "not-found" : "load");
      }
    })();
  }, [sourceID]);

  const typeCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const page of pages ?? []) counts.set(page.type, (counts.get(page.type) ?? 0) + 1);
    return [...counts].sort(([a], [b]) => a.localeCompare(b));
  }, [pages]);
  const visiblePages = useMemo(
    () => pages?.filter((page) => selectedType === "all" || page.type === selectedType) ?? [],
    [pages, selectedType],
  );

  async function copyMCPURL() {
    if (!config) return;
    try {
      await navigator.clipboard.writeText(config.mcp_url);
      setCopyState("copied");
    } catch {
      setCopyState("failed");
    }
  }

  if (error) {
    return (
      <StatusPage
        {...props}
        label={error === "not-found" ? "404" : "Connection error"}
        title={error === "not-found" ? "脳が見つかりません" : "脳を読み込めません"}
      />
    );
  }

  return (
    <main className="shell">
      <SiteHeader {...props} />

      <header className="repo-header">
        <a className="back-link" href="/">
          ← all brains
        </a>
        <p className="eyebrow">Knowledge repository</p>
        <div className="repo-title">
          <div>
            <h1>{brain?.name ?? sourceID}</h1>
            <p className="repo-source mono">{sourceID}</p>
          </div>
          {pages && <span className="mono">{pages.length.toLocaleString("ja-JP")} pages</span>}
        </div>
        {brain?.description && <p className="repo-description">{brain.description}</p>}
      </header>

      {brain && brain.state !== "ready" ? (
        <div className="repository-state" role="status">
          <span className="mono">[{brain.state}]</span>
          <p>この脳はまだ接続できません。状態がreadyになってからページ一覧を開けます。</p>
        </div>
      ) : (
        <>
          <section className="endpoint-line" aria-labelledby="connect-heading">
            <h2 id="connect-heading">MCP endpoint</h2>
            {config ? (
              <>
                <code>{config.mcp_url}</code>
                <button type="button" onClick={() => void copyMCPURL()}>
                  {copyState === "copied" ? "コピー済み" : "コピー"}
                </button>
              </>
            ) : (
              <span className="mono muted">loading…</span>
            )}
          </section>
          {copyState === "failed" && (
            <p className="copy-error">コピーできません。URLを選択してコピーしてください。</p>
          )}

          {pages === null ? (
            <Loading />
          ) : (
            <div className="repository-layout">
              <aside className="type-index" aria-label="ページタイプで絞り込む">
                <p className="eyebrow">Type tree</p>
                <div className="type-tree">
                  <button
                    className={selectedType === "all" ? "tree-root selected" : "tree-root"}
                    type="button"
                    aria-pressed={selectedType === "all"}
                    onClick={() => setSelectedType("all")}
                  >
                    <span>{sourceID}/</span>
                    <span>{pages.length}</span>
                  </button>
                  {typeCounts.map(([type, count], index) => (
                    <button
                      className={selectedType === type ? "tree-branch selected" : "tree-branch"}
                      type="button"
                      aria-pressed={selectedType === type}
                      onClick={() => setSelectedType(type)}
                      key={type}
                    >
                      <span>
                        {index === typeCounts.length - 1 ? "└──" : "├──"} {type}
                      </span>
                      <span>{count}</span>
                    </button>
                  ))}
                </div>
              </aside>

              <section className="page-index" aria-labelledby="page-list-heading">
                <div className="list-heading">
                  <h2 id="page-list-heading">Page index</h2>
                  <span className="mono">{visiblePages.length} entries</span>
                </div>

                {pages.length === 0 && <Empty message="公開ページがありません。ページが追加されると索引が表示されます。" />}
                {pages.length > 0 && visiblePages.length === 0 && (
                  <Empty message="選択したtypeにはページがありません。別のtypeを選択してください。" />
                )}
                {visiblePages.length > 0 && (
                  <div className="page-list">
                    {visiblePages.map((page) => (
                      <article className="page-row" key={page.slug}>
                        <span className="page-type">{page.type}</span>
                        <div className="page-identity">
                          <h3>{page.title}</h3>
                          <p>{page.slug}</p>
                        </div>
                        <time dateTime={page.updated_at}>{formatDate(page.updated_at)}</time>
                      </article>
                    ))}
                  </div>
                )}
              </section>
            </div>
          )}
        </>
      )}
    </main>
  );
}

function AuthPage({ mode, ...props }: PageProps & { mode: "login" | "register" }) {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const isRegister = mode === "register";

  useEffect(() => {
    document.title = `${isRegister ? "新規登録" : "ログイン"} — brainhub`;
  }, [isRegister]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      const email = String(data.get("email"));
      const password = String(data.get("password"));
      if (isRegister) {
        await register({ email, password, name: String(data.get("name")) });
        window.location.assign("/brains/new");
      } else {
        await login({ email, password });
        window.location.assign("/");
      }
    } catch (cause) {
      setError(authErrorMessage(cause, mode));
      setSubmitting(false);
    }
  }

  if (props.viewer) {
    return <StatusPage {...props} label="Signed in" title={`${props.viewer.name} としてログイン中です`} />;
  }

  return (
    <main className="shell">
      <SiteHeader {...props} />
      <div className="form-layout">
        <header className="form-intro">
          <p className="eyebrow">{isRegister ? "Create account" : "Welcome back"}</p>
          <h1>{isRegister ? "新規登録" : "ログイン"}</h1>
          <p>{isRegister ? "領域ごとの脳を作り、AIへ接続する。" : "所有している脳と作成中の状態を確認する。"}</p>
        </header>

        <form className="editor-form" onSubmit={(event) => void submit(event)}>
          {isRegister && (
            <label>
              <span>名前</span>
              <input name="name" autoComplete="name" maxLength={80} required />
            </label>
          )}
          <label>
            <span>メールアドレス</span>
            <input name="email" type="email" autoComplete="email" required />
          </label>
          <label>
            <span>パスワード</span>
            <input
              name="password"
              type="password"
              autoComplete={isRegister ? "new-password" : "current-password"}
              minLength={12}
              maxLength={72}
              required
            />
            {isRegister && <small>12〜72バイト</small>}
          </label>
          {error && <p className="form-error" role="alert">{error}</p>}
          <button className="primary-action" type="submit" disabled={submitting}>
            {submitting ? "処理中…" : isRegister ? "アカウントを作る" : "ログイン"}
          </button>
          <p className="form-switch">
            {isRegister ? "すでにアカウントがある場合は" : "初めて使う場合は"}{" "}
            <a href={isRegister ? "/login" : "/register"}>{isRegister ? "ログイン" : "新規登録"}</a>
          </p>
        </form>
      </div>
    </main>
  );
}

function CreateBrainPage(props: PageProps) {
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    document.title = "脳を作る — brainhub";
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    const data = new FormData(event.currentTarget);
    try {
      const brain = await createBrain({
        source_id: String(data.get("source_id")),
        name: String(data.get("name")),
        description: String(data.get("description")),
        visibility: String(data.get("visibility")) as "public" | "private",
      });
      window.location.assign(`/brains/${encodeURIComponent(brain.source_id)}`);
    } catch (cause) {
      setError(createBrainErrorMessage(cause));
      setSubmitting(false);
    }
  }

  if (props.viewer === undefined) {
    return (
      <main className="shell">
        <SiteHeader {...props} />
        <Loading />
      </main>
    );
  }
  if (!props.viewer) {
    return <StatusPage {...props} label="Authentication required" title="脳を作るにはログインが必要です" actionHref="/login" actionLabel="ログイン" />;
  }

  return (
    <main className="shell">
      <SiteHeader {...props} />
      <div className="form-layout create-layout">
        <header className="form-intro">
          <p className="eyebrow">New repository</p>
          <h1>脳を作る</h1>
          <p>ひとつの領域に、ひとつの脳。source IDは引用とURLに使われ、作成後も変わらない。</p>
        </header>

        <form className="editor-form" onSubmit={(event) => void submit(event)}>
          <label>
            <span>Source ID</span>
            <input
              className="mono"
              name="source_id"
              pattern="[a-z0-9-]{1,32}"
              maxLength={32}
              placeholder="product-research"
              required
            />
            <small>小文字・数字・ハイフン、1〜32文字。defaultは使用不可。</small>
          </label>
          <label>
            <span>名前</span>
            <input name="name" maxLength={80} placeholder="Product research" required />
          </label>
          <label>
            <span>説明</span>
            <textarea name="description" maxLength={500} rows={5} />
          </label>
          <label>
            <span>公開範囲</span>
            <select name="visibility" defaultValue="private">
              <option value="private">private — 所有者だけ</option>
              <option value="public">public — 誰でも閲覧可能</option>
            </select>
          </label>
          {error && <p className="form-error" role="alert">{error}</p>}
          <button className="primary-action" type="submit" disabled={submitting}>
            {submitting ? "作成中…" : "脳を作る"}
          </button>
        </form>
      </div>
    </main>
  );
}

function SiteHeader({ viewer, sessionUnavailable, onLogout }: PageProps) {
  return (
    <header className="site-header">
      <a className="brand" href="/">brainhub</a>
      <nav aria-label="アカウント">
        {sessionUnavailable && <span className="mono">session unavailable</span>}
        {!sessionUnavailable && viewer === null && (
          <>
            <a href="/login">ログイン</a>
            <a href="/register">新規登録</a>
          </>
        )}
        {viewer && (
          <>
            <span className="viewer-name">{viewer.name}</span>
            <a href="/brains/new">脳を作る</a>
            <button type="button" onClick={() => void onLogout()}>ログアウト</button>
          </>
        )}
      </nav>
    </header>
  );
}

function StatusPage({
  label,
  title,
  actionHref = "/",
  actionLabel = "脳の一覧へ戻る",
  ...props
}: PageProps & { label: string; title: string; actionHref?: string; actionLabel?: string }) {
  return (
    <main className="shell">
      <SiteHeader {...props} />
      <div className="status-page">
        <p className="eyebrow">{label}</p>
        <h1>{title}</h1>
        <p>URLか接続状態を確認して、次の操作へ進んでください。</p>
        <a className="back-link" href={actionHref}>← {actionLabel}</a>
      </div>
    </main>
  );
}

function Loading() {
  return <div className="state-panel mono">loading repository…</div>;
}

function Empty({ message }: { message: string }) {
  return <div className="state-panel">{message}</div>;
}

function ErrorPanel({ message }: { message: string }) {
  return <div className="state-panel error-panel" role="alert">{message}</div>;
}

function authErrorMessage(error: unknown, mode: "login" | "register") {
  if (!(error instanceof APIError)) return "接続できません。時間を置いてもう一度お試しください。";
  if (error.status === 429) return "ログイン試行が多すぎます。15分後にもう一度お試しください。";
  if (error.status === 401) return "メールアドレスまたはパスワードが一致しません。";
  if (error.status === 409) return "このメールアドレスはすでに登録されています。";
  if (error.status === 400) return mode === "register" ? "入力内容を確認してください。" : "メールアドレスとパスワードを入力してください。";
  return "処理を完了できません。時間を置いてもう一度お試しください。";
}

function createBrainErrorMessage(error: unknown) {
  if (!(error instanceof APIError)) return "接続できません。時間を置いてもう一度お試しください。";
  if (error.status === 401) return "セッションが切れました。もう一度ログインしてください。";
  if (error.status === 409) return "このsource IDはすでに使われています。";
  if (error.status === 502) return "脳の登録に失敗しました。一覧で状態を確認してください。";
  if (error.status === 400) return "Source ID、名前、説明、公開範囲を確認してください。";
  return "脳を作成できません。時間を置いてもう一度お試しください。";
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(value));
}

export default App;
