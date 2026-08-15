import { useEffect, useMemo, useState } from "react";
import {
  APIError,
  getPublicConfig,
  listBrains,
  listPages,
  type Brain,
  type Page,
  type PublicConfig,
} from "./api";

function App() {
  const match = window.location.pathname.match(/^\/brains\/([^/]+)\/?$/);
  if (match) {
    return <BrainDetailPage sourceID={match[1]} />;
  }
  if (window.location.pathname === "/" || window.location.pathname === "") {
    return <BrainListPage />;
  }
  return <StatusPage label="404" title="ページが見つかりません" />;
}

function BrainListPage() {
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
      <SiteHeader />

      <header className="page-intro">
        <div>
          <p className="eyebrow">Public knowledge repositories</p>
          <h1>Brains</h1>
        </div>
        <p>AIクライアントへ接続できる、領域ごとの知識リポジトリ。</p>
      </header>

      <section aria-labelledby="brain-list-heading">
        <div className="list-heading">
          <h2 id="brain-list-heading">Public sources</h2>
          {brains && <span className="mono">{brains.length} sources</span>}
        </div>

        {error && <ErrorPanel message="脳の一覧を取得できません。接続を確認して再読み込みしてください。" />}
        {!error && brains === null && <Loading />}
        {brains?.length === 0 && <Empty message="公開された脳がありません。新しいsourceが公開されると、ここに表示されます。" />}
        {brains && brains.length > 0 && (
          <div className="brain-table">
            <div className="brain-columns" aria-hidden="true">
              <span>source</span>
              <span>pages</span>
              <span>last sync</span>
              <span />
            </div>
            {brains.map((brain) => (
              <a className="brain-row" href={`/brains/${encodeURIComponent(brain.id)}`} key={brain.id}>
                <strong>{brain.id}</strong>
                <span className="mono">{brain.page_count.toLocaleString("ja-JP")}</span>
                <time className="mono" dateTime={brain.last_sync_at ?? undefined}>
                  {formatDate(brain.last_sync_at)}
                </time>
                <span className="row-arrow" aria-hidden="true">
                  →
                </span>
              </a>
            ))}
          </div>
        )}
      </section>
    </main>
  );
}

function BrainDetailPage({ sourceID }: { sourceID: string }) {
  const [pages, setPages] = useState<Page[] | null>(null);
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [selectedType, setSelectedType] = useState("all");
  const [error, setError] = useState<"not-found" | "load" | null>(null);
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");

  useEffect(() => {
    document.title = `${sourceID} — brainhub`;
    void Promise.all([listPages(sourceID), getPublicConfig()])
      .then(([loadedPages, loadedConfig]) => {
        setPages(loadedPages);
        setConfig(loadedConfig);
      })
      .catch((cause: unknown) => {
        setError(cause instanceof APIError && cause.status === 404 ? "not-found" : "load");
      });
  }, [sourceID]);

  const typeCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const page of pages ?? []) {
      counts.set(page.type, (counts.get(page.type) ?? 0) + 1);
    }
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
        label={error === "not-found" ? "404" : "Connection error"}
        title={error === "not-found" ? "脳が見つかりません" : "脳を読み込めません"}
      />
    );
  }

  return (
    <main className="shell">
      <SiteHeader />

      <header className="repo-header">
        <a className="back-link" href="/">
          ← all brains
        </a>
        <p className="eyebrow">Public brain</p>
        <div className="repo-title">
          <h1>{sourceID}</h1>
          {pages && <span className="mono">{pages.length.toLocaleString("ja-JP")} pages</span>}
        </div>
      </header>

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

            {pages.length === 0 && <Empty message="公開ページがありません。ページが追加されると、ここに索引が表示されます。" />}
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
    </main>
  );
}

function SiteHeader() {
  return (
    <header className="site-header">
      <a className="brand" href="/">
        brainhub
      </a>
      <span className="mono">knowledge repositories</span>
    </header>
  );
}

function StatusPage({ label, title }: { label: string; title: string }) {
  return (
    <main className="shell status-page">
      <p className="eyebrow">{label}</p>
      <h1>{title}</h1>
      <p>URLか接続状態を確認して、脳の一覧からもう一度選択してください。</p>
      <a className="back-link" href="/">
        ← 脳の一覧へ戻る
      </a>
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
  return <div className="state-panel error-panel">{message}</div>;
}

function formatDate(value: string | null) {
  if (!value) return "—";
  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(value));
}

export default App;
