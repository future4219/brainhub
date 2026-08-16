import { Link } from "react-router-dom";

import type { ConnectSectionProps } from "@/components/features/BrainDetail/types";
import { copyLabel } from "@/components/features/BrainDetail/utils";
import { Button, buttonVariants } from "@/components/ui/Button";
import { CopyValue } from "@/components/ui/CopyValue";
import { Feedback } from "@/components/ui/Feedback";
import { Panel } from "@/components/ui/Panel";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { StateBadge } from "@/components/ui/StateBadge";
import { brainUrl, loginUrl } from "@/config/url";
import { connectionSteps } from "@/lib/format";
import { cn } from "@/lib/utils";

export function ConnectSection(props: ConnectSectionProps) {
  const activeClient =
    props.clients?.find((client) => client.state === "active") ?? null;
  const steps = props.config
    ? connectionSteps(
        props.client,
        props.config.mcp_url,
        activeClient?.client_id ?? null,
        props.sourceID,
      )
    : [];

  return (
    <div className="mt-6">
      <header className="mb-6">
        <h2 className="text-section font-semibold">
          {props.sourceID} に繋ぐ
        </h2>
        <p className="mt-2 max-w-copy text-body leading-copy text-text-secondary">
          下の 2 つの値をクライアントに渡す。URL
          を貼るだけでは繋がらないので、先にそこだけ読んでほしい。
        </p>
      </header>
      {props.error && (
        <Feedback kind="error">{props.error}</Feedback>
      )}
      {!props.config && !props.error && (
        <Feedback kind="loading">loading connection…</Feedback>
      )}
      {props.config && (
        <div className="flex flex-col items-start gap-6 lg:flex-row">
          <div className="min-w-0 flex-1 space-y-6">
            <Panel className="border-border-strong bg-surface p-4">
              <h3 className="text-body font-semibold">最初の接続は失敗する</h3>
              <p className="mt-2 text-ui leading-prose text-text-tertiary">
                クライアントは自動登録（Dynamic Client
                Registration）を試みるが、brainhub は対応していない。URL
                を貼った直後にこのエラーが出る。
              </p>
              <code className="mt-3 block rounded-control border border-border-control bg-canvas px-3 py-2 text-sm leading-copy text-text-code">
                コネクタを編集して OAuth クライアント ID を追加してください
              </code>
              <p className="mt-3 text-ui leading-prose text-text-tertiary">
                エラーを見てから手順 03 に進めばよい。やり直しは不要。
              </p>
            </Panel>
            <Panel>
              <div className="flex flex-wrap items-center gap-2 border-b border-border px-4 py-3">
                <span className="mr-2 font-mono text-xs uppercase tracking-label text-text-muted">
                  client
                </span>
                {(["claude", "chatgpt", "codex"] as const).map((client) => (
                  <button
                    className={cn(
                      "h-control-sm rounded-control border px-3 text-ui",
                      props.client === client
                        ? "border-border-strong bg-surface-selected text-text"
                        : "border-transparent text-text-muted hover:text-text",
                    )}
                    type="button"
                    aria-pressed={props.client === client}
                    onClick={() => props.onClientChange(client)}
                    key={client}
                  >
                    {client === "claude"
                      ? "Claude"
                      : client === "chatgpt"
                        ? "ChatGPT"
                        : "Codex"}
                  </button>
                ))}
              </div>
              <ol>
                {steps.map((step, index) => (
                  <li
                    className="step-grid grid gap-4 border-b border-divider p-4 last:border-b-0"
                    key={step.number}
                  >
                    <span className="font-mono text-sm text-text-muted">
                      {step.number}
                    </span>
                    <div className="min-w-0">
                      <h3 className="text-body font-semibold">{step.title}</h3>
                      <p className="mt-2 text-ui leading-prose text-text-secondary">
                        {step.body}
                      </p>
                      {step.code && (
                        <div className="mt-3 flex overflow-hidden rounded-control border border-border-control bg-canvas">
                          <pre className="min-w-0 flex-1 overflow-x-auto whitespace-pre p-3 font-mono text-sm leading-prose text-text-code">
                            {step.code}
                          </pre>
                          <Button
                            className="h-auto rounded-none border-0 border-l border-border-control"
                            variant="ghost"
                            size="sm"
                            type="button"
                            onClick={() =>
                              props.onCopy(
                                `step-${props.client}-${index}`,
                                step.code!,
                              )
                            }
                          >
                            {copyLabel(
                              props.copyState[
                                `step-${props.client}-${index}`
                              ],
                            )}
                          </Button>
                        </div>
                      )}
                      {step.hint && (
                        <p className="mt-3 font-mono text-xs leading-prose text-text-secondary">
                          {step.hint}
                        </p>
                      )}
                    </div>
                  </li>
                ))}
              </ol>
              <div className="p-4">
                <h3 className="text-body font-semibold">繋がったかの確認</h3>
                <p className="mt-2 text-ui leading-prose text-text-secondary">
                  クライアントのツール一覧に次のツールが出ていれば完了。出ていなければ手順
                  03 の Client ID を確認する。
                </p>
                <div className="mt-3 flex flex-wrap gap-2">
                  {[
                    "brainhub_search_pages",
                    "brainhub_get_page",
                    "brainhub_list_types",
                  ].map((tool) => (
                    <code
                      className="rounded-pill border border-border-control bg-surface px-3 py-1 text-xs text-text-code"
                      key={tool}
                    >
                      {tool}
                    </code>
                  ))}
                </div>
              </div>
            </Panel>
          </div>

          <aside className="w-full shrink-0 space-y-4 lg:sticky lg:top-sticky lg:max-w-credentials">
            <Panel>
              <SectionHeading eyebrow="credentials" title="接続情報" />
              <div className="px-4">
                <CopyValue
                  label="MCP URL"
                  value={props.config.mcp_url}
                  note="/api/config が返した値。ブラウザのURLから組み立てない。"
                  copyLabel={copyLabel(props.copyState.mcp)}
                  onCopy={() => props.onCopy("mcp", props.config!.mcp_url)}
                />
                {activeClient && activeClient.client_id && (
                  <CopyValue
                    label="OAuth Client ID"
                    value={activeClient.client_id}
                    note="public client。シークレットは不要。"
                    copyLabel={copyLabel(props.copyState.client)}
                    onCopy={() =>
                      props.onCopy("client", activeClient.client_id!)
                    }
                  />
                )}
              </div>
            </Panel>
            {props.viewer && props.authorized && (
              <Panel>
                <SectionHeading
                  eyebrow="issued clients"
                  title="接続クライアント"
                  meta={
                    <Button
                      size="sm"
                      type="button"
                      disabled={props.submitting}
                      onClick={props.onIssue}
                    >
                      {props.submitting ? "発行中…" : "発行"}
                    </Button>
                  }
                />
                {props.clients === null && (
                  <Feedback kind="loading">loading clients…</Feedback>
                )}
                {props.clients?.length === 0 && (
                  <Feedback kind="empty">
                    発行済み Client ID
                    はありません。上の「発行」から作成してください。
                  </Feedback>
                )}
                {props.clients?.map((client) => (
                  <div
                    className="border-b border-divider p-4 last:border-b-0"
                    key={client.id}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <StateBadge state={client.state} />
                      <Button
                        variant="outline"
                        size="sm"
                        type="button"
                        disabled={
                          client.state === "revoked" || client.client_id === null
                        }
                        onClick={() => props.onRevoke(client.id)}
                      >
                        失効
                      </Button>
                    </div>
                    <p className="mt-2 break-words font-mono text-xs text-text-muted">
                      {client.client_id ?? "client id pending"}
                    </p>
                    {client.state_reason && (
                      <p className="mt-2 text-xs text-text-secondary">
                        {client.state_reason}
                      </p>
                    )}
                  </div>
                ))}
              </Panel>
            )}
            {!props.viewer && (
              <Panel className="p-4">
                <p className="text-ui text-text-secondary">
                  Client IDを発行するにはログインしてください。
                </p>
                <Link
                  className={`${buttonVariants({ size: "sm" })} mt-4`}
                  to={loginUrl(brainUrl(props.sourceID, "connect"))}
                >
                  ログイン
                </Link>
              </Panel>
            )}
            {props.viewer && props.authorized === false && (
              <Feedback kind="empty">
                この脳のClient
                IDを発行する権限がありません。所有者へ招待を依頼してください。
              </Feedback>
            )}
            <Panel>
              <SectionHeading eyebrow="access" title="公開範囲" />
              <div className="space-y-3 p-4 text-ui leading-copy text-text-secondary">
                <p>
                  <strong className="text-text">
                    {props.brain?.visibility}
                  </strong>{" "}
                  —{` `}
                  {props.brain?.visibility === "public"
                    ? "誰でも接続できる。読み取りのみ。"
                    : "招待されたアカウントだけが接続できる。"}
                </p>
                <p>
                  <strong className="text-text">
                    {props.pageCount ?? "—"} pages
                  </strong>{" "}
                  — 全ページが検索・取得の対象。
                </p>
                <p>
                  <strong className="text-text">slug</strong> — 不変の引用キー。
                </p>
              </div>
            </Panel>
            {(props.copyState.mcp === "failed" ||
              props.copyState.client === "failed") && (
              <Feedback kind="error">
                コピーできません。値を選択してコピーしてください。
              </Feedback>
            )}
          </aside>
        </div>
      )}
    </div>
  );
}
