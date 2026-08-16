import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";

import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

import {
  APIError,
  acceptInvitation,
  createBrain,
  createInvitation,
  getBrain,
  getCurrentUser,
  getInvitation,
  getPublicConfig,
  issueClient,
  listBrains,
  listClients,
  listInvitations,
  listPages,
  login,
  logout,
  register,
  revokeClient,
  revokeInvitation,
  type Brain,
  type Invitation,
  type IssuedClient,
  type Page,
  type PublicConfig,
  type User,
} from "./api";

const shellClass = "mx-auto min-h-screen w-[min(calc(100%_-_2rem),72.5rem)] pb-20 sm:w-[min(calc(100%_-_3rem),72.5rem)]";
const eyebrowClass = "mb-2 font-mono text-[0.68rem] font-semibold uppercase tracking-[0.14em] text-muted";
const linkClass = "underline decoration-rule underline-offset-4 hover:decoration-signal focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-signal";
const sectionHeadingClass = "flex min-h-14 items-center justify-between gap-6 border-b border-ink";
const fieldClass = "grid gap-2 border-b border-rule py-5";
const fieldLabelClass = "text-[0.78rem] font-semibold";
const hintClass = "text-[0.7rem] leading-5 text-muted";
const selectClass = "h-10 w-full rounded-[1px] border border-rule bg-paper px-3 text-sm text-ink outline-none focus-visible:border-signal focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-signal";
const errorClass = "mt-4 bg-selection px-3.5 py-3 text-[0.8rem] leading-6 text-ink";

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

  const inviteMatch = path.match(/^\/invite\/([^/]+)\/?$/);
  if (inviteMatch) return <InvitePage {...shared} token={inviteMatch[1]} />;

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
    <main className={shellClass}>
      <SiteHeader {...props} />

      <header className="grid items-end gap-7 py-14 md:grid-cols-[minmax(0,1fr)_minmax(16rem,26rem)] md:gap-12 md:py-20">
        <div>
          <p className={eyebrowClass}>Knowledge repositories</p>
          <h1 className="text-[clamp(3.5rem,8vw,6.5rem)] font-semibold leading-[0.9] tracking-[-0.06em]">Brains</h1>
        </div>
        <p className="mb-1 max-w-[38ch] leading-8 text-muted">中身を確かめ、必要な知識だけを普段使っているAIへ接続する。</p>
      </header>

      <section aria-labelledby="brain-list-heading">
        <div className={sectionHeadingClass}>
          <h2 id="brain-list-heading" className="text-[0.94rem] font-semibold">
            {props.viewer ? "Accessible brains" : "Public brains"}
          </h2>
          {brains && <span className="font-mono text-xs text-muted">{brains.length} repositories</span>}
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
          <div>
            <div
              className="hidden min-h-10 grid-cols-[minmax(0,1fr)_6rem_9rem_1.5rem] items-center gap-5 font-mono text-[0.66rem] uppercase tracking-[0.08em] text-muted sm:grid"
              aria-hidden="true"
            >
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
  const available = brain.state === "ready";
  const rowClass = cn(
    "grid min-h-20 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2 border-b border-rule py-4 text-sm sm:grid-cols-[minmax(0,1fr)_6rem_9rem_1.5rem] sm:gap-5",
    available ? "hover:bg-selection focus-visible:bg-selection focus-visible:outline-2 focus-visible:outline-signal" : "text-muted",
  );
  const content = (
    <>
      <div className="min-w-0">
        <strong className="block text-[1.02rem] font-semibold text-ink">{brain.name}</strong>
        <span className="font-mono text-xs text-muted">{brain.source_id}</span>
        {brain.description && <p className="mt-1.5 max-w-[66ch] truncate text-[0.78rem] leading-6 text-muted">{brain.description}</p>}
      </div>
      <span className="self-start font-mono text-xs text-muted sm:self-auto">{brain.visibility}</span>
      <span className="font-mono text-xs text-muted sm:col-auto">[{brain.state}]</span>
      <span className="text-right font-mono text-signal" aria-hidden="true">
        {available ? "→" : "—"}
      </span>
    </>
  );

  if (!available) {
    return (
      <div className={rowClass} aria-disabled="true">
        {content}
      </div>
    );
  }
  return (
    <a className={rowClass} href={`/brains/${encodeURIComponent(brain.source_id)}`}>
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
    return [...counts].sort(([typeA, countA], [typeB, countB]) => countB - countA || typeA.localeCompare(typeB));
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
    <main className={shellClass}>
      <SiteHeader {...props} />

      <header className="py-9 md:pb-12">
        <a className={cn(linkClass, "mb-12 inline-block font-mono text-xs text-muted md:mb-16")} href="/">
          ← all brains
        </a>
        <p className={eyebrowClass}>Knowledge repository</p>
        <div className="items-end justify-between gap-8 sm:flex">
          <div className="min-w-0">
            <h1 className="[overflow-wrap:anywhere] text-[clamp(3rem,8vw,6rem)] font-semibold leading-[0.92] tracking-[-0.06em]">
              {brain?.name ?? sourceID}
            </h1>
            <p className="mt-3 font-mono text-xs text-muted">{sourceID}</p>
          </div>
          {pages && <span className="mt-4 block shrink-0 pb-1 font-mono text-xs text-muted sm:mt-0">{pages.length.toLocaleString("ja-JP")} pages</span>}
        </div>
        {brain?.description && <p className="mt-7 max-w-[68ch] leading-7 text-muted">{brain.description}</p>}
      </header>

      {brain && brain.state !== "ready" ? (
        <div className="grid gap-3 border-y border-rule py-5 text-muted sm:grid-cols-[9.5rem_minmax(0,1fr)]" role="status">
          <span className="font-mono text-xs text-ink">[{brain.state}]</span>
          <p className="leading-6">この脳はまだ接続できません。状態がreadyになってからページ一覧を開けます。</p>
        </div>
      ) : (
        <>
          <section className="grid min-h-16 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 border-y border-ink py-3 sm:grid-cols-[9.5rem_minmax(0,1fr)_auto] sm:gap-5" aria-labelledby="connect-heading">
            <h2 id="connect-heading" className="col-span-2 font-mono text-xs font-semibold uppercase tracking-[0.08em] sm:col-span-1">
              MCP endpoint
            </h2>
            {config ? (
              <>
                <code className="min-w-0 overflow-x-auto whitespace-nowrap font-mono text-xs text-muted">{config.mcp_url}</code>
                <Button size="sm" type="button" onClick={() => void copyMCPURL()}>
                  {copyState === "copied" ? "コピー済み" : "コピー"}
                </Button>
              </>
            ) : (
              <span className="font-mono text-xs text-muted">loading…</span>
            )}
          </section>
          {copyState === "failed" && <p className={errorClass}>コピーできません。URLを選択してコピーしてください。</p>}

          {props.viewer && config && <ConnectionsSection sourceID={sourceID} config={config} focus={window.location.search === "?connect=1"} />}
          {props.viewer && <InvitationSection sourceID={sourceID} />}

          {pages === null ? (
            <Loading />
          ) : (
            <div className="grid gap-12 pt-14 lg:grid-cols-[15rem_minmax(0,1fr)] lg:gap-14">
              <aside className="self-start lg:sticky lg:top-6" aria-label="ページタイプで絞り込む">
                <p className={eyebrowClass}>Type tree</p>
                <div className="font-mono text-xs">
                  <TypeTreeButton selected={selectedType === "all"} onClick={() => setSelectedType("all")}>
                    <span className="truncate">{sourceID}/</span>
                    <span className="text-right text-[0.7rem] tabular-nums">{pages.length}</span>
                  </TypeTreeButton>
                  {typeCounts.map(([type, count], index) => (
                    <TypeTreeButton selected={selectedType === type} onClick={() => setSelectedType(type)} key={type} branch>
                      <span className="truncate">
                        {index === typeCounts.length - 1 ? "└──" : "├──"} {type}
                      </span>
                      <span className="text-right text-[0.7rem] tabular-nums">{count}</span>
                    </TypeTreeButton>
                  ))}
                </div>
              </aside>

              <section className="min-w-0" aria-labelledby="page-list-heading">
                <div className={sectionHeadingClass}>
                  <h2 id="page-list-heading" className="text-[0.94rem] font-semibold">Page index</h2>
                  <span className="font-mono text-xs text-muted">{visiblePages.length} entries</span>
                </div>

                {pages.length === 0 && <Empty message="公開ページがありません。ページが追加されると索引が表示されます。" />}
                {pages.length > 0 && visiblePages.length === 0 && <Empty message="選択したtypeにはページがありません。別のtypeを選択してください。" />}
                {visiblePages.length > 0 && (
                  <div>
                    {visiblePages.map((page) => (
                      <article className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-2 border-b border-rule py-4 sm:grid-cols-[7rem_minmax(0,1fr)_6rem] sm:gap-5" key={page.slug}>
                        <span className="truncate pt-0.5 font-mono text-[0.68rem] font-medium text-signal">{page.type}</span>
                        <div className="col-span-2 min-w-0 sm:col-span-1">
                          <h3 className="mb-1 text-[0.96rem] font-medium leading-6">{page.title}</h3>
                          <p className="[overflow-wrap:anywhere] font-mono text-[0.7rem] leading-5 text-muted">{page.slug}</p>
                        </div>
                        <time className="col-start-2 row-start-1 pt-0.5 text-right font-mono text-[0.68rem] text-muted sm:col-start-3" dateTime={page.updated_at}>
                          {formatDate(page.updated_at)}
                        </time>
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

function TypeTreeButton({ selected, branch = false, onClick, children }: { selected: boolean; branch?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      className={cn(
        "grid w-full grid-cols-[minmax(0,1fr)_3.5rem] items-center gap-3 px-3 text-left outline-none transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-signal motion-reduce:transition-none",
        branch ? "min-h-10" : "min-h-11 font-semibold",
        selected ? "bg-signal text-paper" : "bg-transparent text-muted hover:bg-selection hover:text-ink",
      )}
      type="button"
      aria-pressed={selected}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function InvitationSection({ sourceID }: { sourceID: string }) {
  const [invitations, setInvitations] = useState<Invitation[] | null>(null);
  const [authorized, setAuthorized] = useState<boolean | null>(null);
  const [createdLink, setCreatedLink] = useState("");
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function refresh() {
    try {
      setInvitations(await listInvitations(sourceID));
      setAuthorized(true);
    } catch (cause) {
      if (cause instanceof APIError && cause.status === 403) {
        setAuthorized(false);
        return;
      }
      setError("招待一覧を取得できません。");
    }
  }

  useEffect(() => {
    void refresh();
  }, [sourceID]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    setSubmitting(true);
    setError("");
    const data = new FormData(form);
    const expiresAt = new Date(String(data.get("expires_at")));
    try {
      const invitation = await createInvitation(sourceID, {
        email: String(data.get("email")),
        role: String(data.get("role")) as "owner" | "editor" | "reader",
        expires_at: expiresAt.toISOString(),
      });
      setCreatedLink(`${window.location.origin}/invite/${encodeURIComponent(invitation.token)}`);
      setCopyState("idle");
      form.reset();
      await refresh();
    } catch (cause) {
      setError(cause instanceof APIError && cause.status === 400 ? "メール、role、有効期限を確認してください。" : "招待を作成できません。");
    } finally {
      setSubmitting(false);
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(createdLink);
      setCopyState("copied");
    } catch {
      setCopyState("failed");
    }
  }

  async function revoke(id: string) {
    try {
      await revokeInvitation(id);
      await refresh();
    } catch {
      setError("招待を失効できません。");
    }
  }

  if (authorized !== true) return null;
  return (
    <section className="mt-14 border-y border-ink" aria-labelledby="invitation-heading">
      <AccessHeading eyebrow="Repository access" title="Invite" id="invitation-heading">
        <span className="font-mono text-[0.68rem] text-muted">owner only</span>
      </AccessHeading>
      <form className="grid items-end gap-4 border-t border-rule py-5 md:grid-cols-[minmax(12rem,1fr)_8rem_minmax(12rem,0.8fr)_auto]" onSubmit={(event) => void submit(event)}>
        <label className="grid gap-2">
          <span className="font-mono text-[0.66rem] uppercase tracking-[0.04em] text-muted">Email constraint</span>
          <Input name="email" type="email" placeholder="空欄なら誰でも受諾可" />
        </label>
        <label className="grid gap-2">
          <span className="font-mono text-[0.66rem] uppercase tracking-[0.04em] text-muted">Role</span>
          <select className={selectClass} name="role" defaultValue="reader">
            <option value="reader">reader</option>
            <option value="editor">editor</option>
            <option value="owner">owner</option>
          </select>
        </label>
        <label className="grid gap-2">
          <span className="font-mono text-[0.66rem] uppercase tracking-[0.04em] text-muted">Expires at</span>
          <Input name="expires_at" type="datetime-local" required />
        </label>
        <Button type="submit" disabled={submitting}>{submitting ? "生成中…" : "招待リンクを生成"}</Button>
      </form>
      {createdLink && (
        <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 border-t border-rule py-4" role="status">
          <code className="overflow-x-auto whitespace-nowrap font-mono text-xs text-ink">{createdLink}</code>
          <Button size="sm" type="button" onClick={() => void copyLink()}>{copyState === "copied" ? "コピー済み" : "コピー"}</Button>
        </div>
      )}
      {copyState === "failed" && <p className={errorClass}>リンクを選択してコピーしてください。</p>}
      {error && <p className={errorClass} role="alert">{error}</p>}
      {invitations && invitations.length > 0 && (
        <div className="border-t border-rule">
          {invitations.map((invitation) => (
            <div className="grid min-h-16 grid-cols-[minmax(0,1fr)_auto] items-center gap-4 border-b border-rule py-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_12rem_auto]" key={invitation.id}>
              <div>
                <strong className="block text-sm">{invitation.email ?? "anyone with link"}</strong>
                <span className="mt-1 block font-mono text-[0.68rem] text-muted">{invitation.role} / [{invitation.state}]</span>
              </div>
              <time className="font-mono text-[0.68rem] text-muted sm:text-right" dateTime={invitation.expires_at}>{formatDateTime(invitation.expires_at)}</time>
              <Button variant="outline" size="sm" type="button" disabled={invitation.state !== "pending"} onClick={() => void revoke(invitation.id)}>revoke</Button>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function ConnectionsSection({ sourceID, config, focus }: { sourceID: string; config: PublicConfig; focus: boolean }) {
  const [clients, setClients] = useState<IssuedClient[] | null>(null);
  const [authorized, setAuthorized] = useState<boolean | null>(null);
  const [issued, setIssued] = useState<IssuedClient | null>(null);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function refresh() {
    try {
      setClients(await listClients(sourceID));
      setAuthorized(true);
    } catch (cause) {
      if (cause instanceof APIError && cause.status === 403) {
        setAuthorized(false);
        return;
      }
      setError("接続クライアントを取得できません。");
    }
  }

  useEffect(() => {
    void refresh();
  }, [sourceID]);

  async function issue() {
    setSubmitting(true);
    setError("");
    try {
      const client = await issueClient(sourceID);
      setIssued(client);
      await refresh();
    } catch {
      setError("Claude用Client IDを発行できません。失敗記録はorphanとして保存されます。");
    } finally {
      setSubmitting(false);
    }
  }

  async function revoke(id: string) {
    try {
      await revokeClient(id);
      await refresh();
    } catch {
      setError("クライアントを失効できません。状態を確認してください。");
    }
  }

  if (authorized !== true) return null;
  const displayed = issued ?? clients?.find((client) => client.state === "active") ?? null;
  return (
    <section className={cn("mt-14 border-y border-ink", focus && "outline-2 outline-offset-8 outline-signal")} aria-labelledby="connection-heading">
      <AccessHeading eyebrow="OAuth connection" title="Connect Claude" id="connection-heading">
        <Button size="sm" type="button" disabled={submitting} onClick={() => void issue()}>{submitting ? "発行中…" : "Client IDを発行"}</Button>
      </AccessHeading>
      {displayed && (
        <dl className="border-t border-rule">
          <ConnectionValue term="MCP URL"><code>{config.mcp_url}</code></ConnectionValue>
          <ConnectionValue term="Client ID"><code>{displayed.client_id ?? "発行未完了"}</code></ConnectionValue>
          <ConnectionValue term="Secret">なし（public client）</ConnectionValue>
        </dl>
      )}
      <div className="grid gap-5 border-t border-rule py-6 text-sm leading-7 text-muted md:grid-cols-[minmax(12rem,0.55fr)_minmax(0,1.45fr)] md:gap-8">
        <p><strong className="text-ink">URLを貼るだけでは接続できません。</strong></p>
        <ol className="list-decimal space-y-2 pl-5">
          <li>Claudeの設定でカスタムコネクタを追加し、上のMCP URLを入力する。</li>
          <li>動的クライアント登録のエラーが出たら、そのコネクタを編集する。</li>
          <li>「OAuthクライアントID」に発行したClient IDを追加して接続し直す。</li>
        </ol>
      </div>
      {error && <p className={errorClass} role="alert">{error}</p>}
      {clients && clients.length > 0 && (
        <div className="border-t border-rule">
          {clients.map((client) => (
            <div className="grid min-h-16 grid-cols-[minmax(0,1fr)_auto] items-center gap-4 border-b border-rule py-3 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_12rem_auto]" key={client.id}>
              <div>
                <strong className="block text-sm">{client.label}</strong>
                <span className="mt-1 block [overflow-wrap:anywhere] font-mono text-[0.68rem] text-muted">[{client.state}] {client.client_id ?? "no client id"}</span>
              </div>
              <time className="font-mono text-[0.68rem] text-muted sm:text-right" dateTime={client.issued_at}>{formatDateTime(client.issued_at)}</time>
              <Button variant="outline" size="sm" type="button" disabled={client.state === "revoked" || client.client_id === null} onClick={() => void revoke(client.id)}>revoke</Button>
              {client.state_reason && <p className="col-span-full [overflow-wrap:anywhere] text-xs text-muted">{client.state_reason}</p>}
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function AccessHeading({ eyebrow, title, id, children }: { eyebrow: string; title: string; id: string; children: ReactNode }) {
  return (
    <header className="flex min-h-20 items-center justify-between gap-6">
      <div>
        <p className={cn(eyebrowClass, "mb-1")}>{eyebrow}</p>
        <h2 id={id} className="text-xl font-semibold">{title}</h2>
      </div>
      {children}
    </header>
  );
}

function ConnectionValue({ term, children }: { term: string; children: ReactNode }) {
  return (
    <div className="grid gap-2 border-b border-rule py-3.5 last:border-b-0 sm:grid-cols-[9.5rem_minmax(0,1fr)] sm:gap-5">
      <dt className="font-mono text-[0.66rem] uppercase tracking-[0.04em] text-muted">{term}</dt>
      <dd className="m-0 [overflow-wrap:anywhere] text-sm">{children}</dd>
    </div>
  );
}

function InvitePage({ token, ...props }: PageProps & { token: string }) {
  const [preview, setPreview] = useState<{ brain_name: string; invited_by_name: string } | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    document.title = "招待 — brainhub";
    void getInvitation(token).then(setPreview).catch(() => setNotFound(true));
  }, [token]);

  async function accept() {
    setSubmitting(true);
    setError("");
    try {
      const accepted = await acceptInvitation(token);
      window.location.assign(`/brains/${encodeURIComponent(accepted.source_id)}?connect=1`);
    } catch (cause) {
      if (cause instanceof APIError && cause.status === 409) setError("この招待はすでに受諾済みです。");
      else if (cause instanceof APIError && cause.status === 403) setError("この招待に指定されたメールアドレスと一致しません。");
      else if (cause instanceof APIError && cause.status === 404) setNotFound(true);
      else setError("招待を受諾できません。");
      setSubmitting(false);
    }
  }

  if (notFound) return <StatusPage {...props} label="404" title="招待が見つかりません" />;
  return (
    <main className={shellClass}>
      <SiteHeader {...props} />
      <div className="w-full max-w-[48rem] py-20 md:py-28">
        <p className={eyebrowClass}>Repository invitation</p>
        <h1 className="[overflow-wrap:anywhere] text-[clamp(3.2rem,9vw,7rem)] font-semibold leading-[0.92] tracking-[-0.06em]">{preview?.brain_name ?? "invitation"}</h1>
        {preview ? <p className="mt-8 leading-8 text-muted"><strong className="text-ink">{preview.invited_by_name}</strong> から脳への招待が届いています。</p> : <Loading />}
        {props.viewer === undefined && <p className="mt-5 font-mono text-xs text-muted">sessionを確認中…</p>}
        {props.viewer === null && preview && (
          <div className="mt-8 flex flex-wrap items-center gap-4">
            <p className="w-full text-sm leading-7 text-muted">受諾するにはログイン、または新規登録してください。完了後にこの招待へ戻ります。</p>
            <a className={buttonVariants()} href={`/login?next=${encodeURIComponent(`/invite/${encodeURIComponent(token)}`)}`}>ログイン</a>
            <a className={cn(linkClass, "text-sm")} href={`/register?next=${encodeURIComponent(`/invite/${encodeURIComponent(token)}`)}`}>新規登録</a>
          </div>
        )}
        {props.viewer && preview && <Button className="mt-8" type="button" disabled={submitting} onClick={() => void accept()}>{submitting ? "受諾中…" : "招待を受諾する"}</Button>}
        {error && <p className={errorClass} role="alert">{error}</p>}
      </div>
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
        window.location.assign(safeNextPath("/brains/new"));
      } else {
        await login({ email, password });
        window.location.assign(safeNextPath("/"));
      }
    } catch (cause) {
      setError(authErrorMessage(cause, mode));
      setSubmitting(false);
    }
  }

  if (props.viewer) return <StatusPage {...props} label="Signed in" title={`${props.viewer.name} としてログイン中です`} />;

  return (
    <main className={shellClass}>
      <SiteHeader {...props} />
      <div className="grid gap-14 py-16 md:grid-cols-[minmax(16rem,0.8fr)_minmax(24rem,1.2fr)] md:gap-[clamp(3.5rem,9vw,7.5rem)] md:py-24">
        <header className="self-start">
          <p className={eyebrowClass}>{isRegister ? "Create account" : "Welcome back"}</p>
          <h1 className="text-[clamp(3rem,7vw,5.5rem)] font-semibold leading-[0.92] tracking-[-0.06em]">{isRegister ? "新規登録" : "ログイン"}</h1>
          <p className="mt-7 max-w-[36ch] leading-8 text-muted">{isRegister ? "領域ごとの脳を作り、AIへ接続する。" : "所有している脳と作成中の状態を確認する。"}</p>
        </header>

        <form className="border-t border-ink" onSubmit={(event) => void submit(event)}>
          {isRegister && (
            <label className={fieldClass}>
              <span className={fieldLabelClass}>名前</span>
              <Input name="name" autoComplete="name" maxLength={80} required />
            </label>
          )}
          <label className={fieldClass}>
            <span className={fieldLabelClass}>メールアドレス</span>
            <Input name="email" type="email" autoComplete="email" required />
          </label>
          <label className={fieldClass}>
            <span className={fieldLabelClass}>パスワード</span>
            <Input name="password" type="password" autoComplete={isRegister ? "new-password" : "current-password"} minLength={12} maxLength={72} required />
            {isRegister && <small className={hintClass}>12〜72バイト</small>}
          </label>
          {error && <p className={errorClass} role="alert">{error}</p>}
          <Button className="mt-7" type="submit" disabled={submitting}>{submitting ? "処理中…" : isRegister ? "アカウントを作る" : "ログイン"}</Button>
          <p className="mt-6 text-[0.78rem] text-muted">
            {isRegister ? "すでにアカウントがある場合は" : "初めて使う場合は"}{" "}
            <a className={linkClass} href={authSwitchHref(isRegister ? "/login" : "/register")}>{isRegister ? "ログイン" : "新規登録"}</a>
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
    return <main className={shellClass}><SiteHeader {...props} /><Loading /></main>;
  }
  if (!props.viewer) {
    return <StatusPage {...props} label="Authentication required" title="脳を作るにはログインが必要です" actionHref="/login" actionLabel="ログイン" />;
  }

  return (
    <main className={shellClass}>
      <SiteHeader {...props} />
      <div className="grid gap-14 py-16 md:grid-cols-[minmax(16rem,0.8fr)_minmax(24rem,1.2fr)] md:gap-[clamp(3.5rem,9vw,7.5rem)] md:py-24">
        <header className="self-start">
          <p className={eyebrowClass}>New repository</p>
          <h1 className="text-[clamp(3rem,7vw,5.5rem)] font-semibold leading-[0.92] tracking-[-0.06em]">脳を作る</h1>
          <p className="mt-7 max-w-[36ch] leading-8 text-muted">ひとつの領域に、ひとつの脳。source IDは引用とURLに使われ、作成後も変わらない。</p>
        </header>

        <form className="border-t border-ink" onSubmit={(event) => void submit(event)}>
          <label className={fieldClass}>
            <span className={fieldLabelClass}>Source ID</span>
            <Input className="font-mono" name="source_id" pattern="[a-z0-9-]{1,32}" maxLength={32} placeholder="product-research" required />
            <small className={hintClass}>小文字・数字・ハイフン、1〜32文字。defaultは使用不可。</small>
          </label>
          <label className={fieldClass}>
            <span className={fieldLabelClass}>名前</span>
            <Input name="name" maxLength={80} placeholder="Product research" required />
          </label>
          <label className={fieldClass}>
            <span className={fieldLabelClass}>説明</span>
            <Textarea name="description" maxLength={500} rows={5} />
          </label>
          <label className={fieldClass}>
            <span className={fieldLabelClass}>公開範囲</span>
            <select className={selectClass} name="visibility" defaultValue="private">
              <option value="private">private — 所有者だけ</option>
              <option value="public">public — 誰でも閲覧可能</option>
            </select>
          </label>
          {error && <p className={errorClass} role="alert">{error}</p>}
          <Button className="mt-7" type="submit" disabled={submitting}>{submitting ? "作成中…" : "脳を作る"}</Button>
        </form>
      </div>
    </main>
  );
}

function SiteHeader({ viewer, sessionUnavailable, onLogout }: PageProps) {
  return (
    <header className="flex min-h-[4.25rem] items-center justify-between border-b border-rule">
      <a className="text-lg font-semibold tracking-[-0.03em] focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-signal" href="/">brainhub</a>
      <nav className="flex items-center gap-3 text-xs text-muted sm:gap-5" aria-label="アカウント">
        {sessionUnavailable && <span className="font-mono">session unavailable</span>}
        {!sessionUnavailable && viewer === null && (
          <>
            <a className={linkClass} href="/login">ログイン</a>
            <a className={linkClass} href="/register">新規登録</a>
          </>
        )}
        {viewer && (
          <>
            <span className="hidden font-semibold text-ink sm:inline">{viewer.name}</span>
            <a className={linkClass} href="/brains/new">脳を作る</a>
            <Button variant="ghost" size="sm" type="button" onClick={() => void onLogout()}>ログアウト</Button>
          </>
        )}
      </nav>
    </header>
  );
}

function StatusPage({ label, title, actionHref = "/", actionLabel = "脳の一覧へ戻る", ...props }: PageProps & { label: string; title: string; actionHref?: string; actionLabel?: string }) {
  return (
    <main className={shellClass}>
      <SiteHeader {...props} />
      <div className="flex min-h-[calc(100vh-4.25rem)] flex-col items-start justify-center">
        <p className={eyebrowClass}>{label}</p>
        <h1 className="max-w-[50rem] text-[clamp(3rem,8vw,6rem)] font-semibold leading-[0.95] tracking-[-0.06em]">{title}</h1>
        <p className="mb-7 mt-6 max-w-[34rem] leading-7 text-muted">URLか接続状態を確認して、次の操作へ進んでください。</p>
        <a className={cn(linkClass, "font-mono text-xs text-muted")} href={actionHref}>← {actionLabel}</a>
      </div>
    </main>
  );
}

function Loading() {
  return <div className="border-b border-rule px-5 py-11 text-center font-mono text-sm text-muted">loading repository…</div>;
}

function Empty({ message }: { message: string }) {
  return <div className="border-b border-rule px-5 py-11 text-center text-sm text-muted">{message}</div>;
}

function ErrorPanel({ message }: { message: string }) {
  return <div className="border-b border-rule bg-selection px-5 py-11 text-center text-sm text-ink" role="alert">{message}</div>;
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

function safeNextPath(fallback: string) {
  const next = new URLSearchParams(window.location.search).get("next");
  return next && /^\/invite\/[^/?#]+$/.test(next) ? next : fallback;
}

function authSwitchHref(path: string) {
  const next = safeNextPath("");
  return next ? `${path}?next=${encodeURIComponent(next)}` : path;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", { year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date(value));
}

function formatDateTime(value: string) {
  return new Intl.DateTimeFormat("ja-JP", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" }).format(new Date(value));
}

export default App;
