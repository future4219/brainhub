import { Link } from "react-router-dom";

import type { ConnectSectionProps } from "@/components/features/BrainDetail/types";
import { copyLabel } from "@/components/features/BrainDetail/utils";
import { Button, buttonVariants } from "@/components/ui/Button";
import { CopyValue } from "@/components/ui/CopyValue";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { StateBadge } from "@/components/ui/StateBadge";
import { brainUrl, loginUrl } from "@/config/url";
import { connectionSteps, formatDate } from "@/lib/format";

const expiringSoonMilliseconds = 30 * 24 * 60 * 60 * 1000;

export function ConnectSection(props: ConnectSectionProps) {
  const clientID = props.connection?.client?.id ?? null;
  const steps = props.config
    ? connectionSteps(props.config.mcp_url, clientID)
    : [];
  const cliConfig = props.config
    ? `[mcp_servers.brainhub]\nurl = "${props.config.mcp_url}"\nbearer_token_env_var = "BRAINHUB_MCP_TOKEN"`
    : "";

  return (
    <div className="mt-6">
      <header className="mb-6">
        <h2 className="text-section font-semibold">あなたの接続</h2>
        <p className="mt-2 max-w-copy text-body leading-copy text-text-secondary">
          この接続1本で、現在見られるすべての脳をAIクライアントから読み取れます。
          脳が増えても繋ぎ直す必要はありません。
        </p>
      </header>

      <Panel className="mb-6 border-border-strong bg-surface p-4">
        <h3 className="text-body font-semibold">新しい接続情報に切り替えてください</h3>
        <p className="mt-2 text-ui leading-prose text-text-secondary">
          以前の脳ごとのClient IDはこのMCP URLでは利用できません。
          下のMCP URLと新しいClient IDでClaudeのコネクタを更新してください。
        </p>
      </Panel>

      {props.error && <Feedback kind="error">{props.error}</Feedback>}
      {!props.config && !props.error && (
        <Feedback kind="loading">loading connection…</Feedback>
      )}
      {props.config && !props.viewer && (
        <Panel className="p-4">
          <p className="text-ui text-text-secondary">
            接続情報を作るにはログインしてください。
          </p>
          <Link
            className={`${buttonVariants({ size: "sm" })} mt-4`}
            to={loginUrl(brainUrl(props.sourceID, "connect"))}
          >
            ログイン
          </Link>
        </Panel>
      )}
      {props.config && props.viewer && !props.connection && !props.error && (
        <Feedback kind="loading">loading your connection…</Feedback>
      )}
      {props.config && props.viewer && props.connection && (
        <div className="flex flex-col items-start gap-6 lg:flex-row">
          <div className="min-w-0 flex-1 space-y-6">
            {props.connection.reader?.state === "orphan" && (
              <Panel className="p-4">
                <StateBadge state="orphan" />
                <h3 className="mt-3 text-body font-semibold">
                  読み取り接続を再発行してください
                </h3>
                <p className="mt-2 text-ui leading-prose text-text-secondary">
                  GBrainのreader作成または範囲更新に失敗したため、現在のリクエストは転送されません。
                  再発行後にClaudeからもう一度呼び出してください。
                </p>
                {props.connection.reader.state_reason && (
                  <p className="mt-3 break-words font-mono text-xs text-text-muted">
                    {props.connection.reader.state_reason}
                  </p>
                )}
                <Button
                  className="mt-4"
                  size="sm"
                  type="button"
                  disabled={props.readerReissuing}
                  onClick={props.onReissueReader}
                >
                  {props.readerReissuing ? "再発行中…" : "読み取り接続を再発行"}
                </Button>
                {props.readerStatus && (
                  <p className="mt-3 text-xs text-text-secondary">
                    {props.readerStatus}
                  </p>
                )}
              </Panel>
            )}

            <Panel>
              <SectionHeading eyebrow="access" title="見られる脳" />
              {props.connection.visible_brains.length === 0 ? (
                <Feedback kind="empty">
                  接続できるreadyの脳がありません。脳を作るか、招待を受けてください。
                </Feedback>
              ) : (
                props.connection.visible_brains.map((brain) => (
                  <div
                    className="flex items-center justify-between gap-4 border-b border-divider p-4 last:border-b-0"
                    key={brain.source_id}
                  >
                    <div className="min-w-0">
                      <p className="truncate font-mono text-sm text-text">
                        {brain.source_id}
                      </p>
                      <p className="mt-1 text-xs text-text-muted">
                        {roleLabel(brain.role)}
                      </p>
                    </div>
                    <StateBadge state={brain.state} />
                  </div>
                ))
              )}
              <p className="border-t border-divider p-4 text-ui text-text-secondary">
                Membershipや公開範囲の変更は、同じ接続の次の呼び出しから反映されます。
              </p>
            </Panel>

            <Panel>
              <SectionHeading eyebrow="cli" title="CLI用トークン" />
              <div className="border-b border-divider p-4 text-ui leading-prose text-text-secondary">
                CodexやClaude Codeへ設定する90日間有効のBearer tokenです。自動更新はされません。
                期限切れ後はMCPが401のtoken_expiredを返すため、新しいトークンを発行して環境変数を置き換えてください。
              </div>
              <form
                className="flex flex-wrap items-end gap-3 border-b border-divider p-4"
                onSubmit={(event) => {
                  event.preventDefault();
                  props.onIssueCLIToken();
                }}
              >
                <FormField
                  className="min-w-0 flex-1 basis-64 border-0 py-0"
                  label="ラベル"
                  hint="どこで使う鍵か分かる名前。例: codex"
                >
                  <Input
                    value={props.cliTokenLabel}
                    maxLength={64}
                    placeholder="codex"
                    required
                    onChange={(event) =>
                      props.onCLITokenLabel(event.target.value)
                    }
                  />
                </FormField>
                <Button
                  type="submit"
                  disabled={
                    props.cliTokenSubmitting || !props.cliTokenLabel.trim()
                  }
                >
                  {props.cliTokenSubmitting ? "発行中…" : "トークンを発行"}
                </Button>
              </form>
              {props.createdCLIToken && (
                <div className="border-b border-divider px-4">
                  <CopyValue
                    label="CLI TOKEN — 今だけ表示"
                    value={props.createdCLIToken.token}
                    note="再読み込みすると表示できません。見失った場合はこのトークンを失効し、新しく発行してください。"
                    copyLabel={copyLabel(props.copyState["cli-token"])}
                    onCopy={() =>
                      props.onCopy("cli-token", props.createdCLIToken!.token)
                    }
                  />
                </div>
              )}
              <div className="border-b border-divider p-4">
                <p className="text-ui leading-prose text-text-secondary">
                  トークンを環境変数
                  <code className="mx-1 font-mono text-text-code">
                    BRAINHUB_MCP_TOKEN
                  </code>
                  に保存し、Codexの設定へ次を追加します。
                </p>
                <div className="mt-3 flex overflow-hidden rounded-control border border-border-control bg-canvas">
                  <pre className="min-w-0 flex-1 overflow-x-auto whitespace-pre p-3 font-mono text-sm leading-prose text-text-code">
                    {cliConfig}
                  </pre>
                  <Button
                    className="h-auto rounded-none border-0 border-l border-border-control"
                    variant="ghost"
                    size="sm"
                    type="button"
                    onClick={() => props.onCopy("cli-config", cliConfig)}
                  >
                    {copyLabel(props.copyState["cli-config"])}
                  </Button>
                </div>
              </div>
              {props.cliTokenError && (
                <Feedback kind="error">{props.cliTokenError}</Feedback>
              )}
              {props.connection.cli_tokens.length === 0 ? (
                <Feedback kind="empty">
                  CLI用トークンはまだありません。上のフォームから、使う端末ごとに発行してください。
                </Feedback>
              ) : (
                <div>
                  {props.connection.cli_tokens.map((token) => {
                    const status = cliTokenStatus(token.expires_at);
                    return (
                      <div
                        className="flex flex-wrap items-center justify-between gap-4 border-b border-divider p-4 last:border-b-0"
                        key={token.id}
                      >
                        <div className="min-w-0 flex-1">
                          <p className="truncate font-mono text-sm text-text">
                            {token.label}
                          </p>
                          <p className="mt-1 font-mono text-xs text-text-muted">
                            発行 {formatDate(token.created_at)} · 期限 {formatDate(token.expires_at)}
                            {status.detail && ` · ${status.detail}`}
                          </p>
                        </div>
                        <span className="state-badge" data-state={status.state}>
                          [{status.label}]
                        </span>
                        <Button
                          variant="outline"
                          size="sm"
                          type="button"
                          disabled={props.cliTokenRevoking === token.id}
                          onClick={() => props.onRevokeCLIToken(token.id)}
                        >
                          {props.cliTokenRevoking === token.id
                            ? "失効中…"
                            : "失効"}
                        </Button>
                      </div>
                    );
                  })}
                </div>
              )}
              {(props.copyState["cli-token"] === "failed" ||
                props.copyState["cli-config"] === "failed") && (
                <Feedback kind="error">
                  コピーできません。値を選択してコピーしてください。
                </Feedback>
              )}
            </Panel>

            <Panel>
              <SectionHeading eyebrow="claude web" title="接続手順" />
              <div className="border-b border-divider p-4 text-ui leading-prose text-text-secondary">
                Dynamic Client Registrationには対応していません。最初の失敗後にClient IDを手入力します。
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
                              props.onCopy(`step-${index}`, step.code!)
                            }
                          >
                            {copyLabel(props.copyState[`step-${index}`])}
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
            </Panel>
          </div>

          <aside className="w-full shrink-0 space-y-4 lg:sticky lg:top-sticky lg:max-w-credentials">
            <Panel>
              <SectionHeading
                eyebrow="credentials"
                title="接続情報"
                meta={
                  !clientID ? (
                    <Button
                      size="sm"
                      type="button"
                      disabled={props.submitting}
                      onClick={props.onIssue}
                    >
                      {props.submitting ? "発行中…" : "Client IDを発行"}
                    </Button>
                  ) : undefined
                }
              />
              <div className="px-4">
                <CopyValue
                  label="MCP URL"
                  value={props.config.mcp_url}
                  note="/api/configが返した公開URL。"
                  copyLabel={copyLabel(props.copyState.mcp)}
                  onCopy={() => props.onCopy("mcp", props.config!.mcp_url)}
                />
                {clientID && (
                  <CopyValue
                    label="OAuth Client ID"
                    value={clientID}
                    note="Claude Web用のpublic client。"
                    copyLabel={copyLabel(props.copyState.client)}
                    onCopy={() => props.onCopy("client", clientID)}
                  />
                )}
                <div className="border-t border-divider py-4">
                  <p className="font-mono text-xs uppercase tracking-label text-text-muted">
                    Secret
                  </p>
                  <p className="mt-2 font-mono text-sm text-text">なし</p>
                </div>
              </div>
            </Panel>

            {props.canReissueWriter && (
              <Panel className="p-4">
                <h3 className="text-body font-semibold">Web書き込み用client</h3>
                <p className="mt-2 text-ui leading-prose text-text-secondary">
                  この脳のwriterがorphanの場合に再発行します。Claudeの読み取り接続とは別です。
                </p>
                <Button
                  className="mt-4"
                  variant="outline"
                  size="sm"
                  type="button"
                  disabled={props.writerReissuing}
                  onClick={props.onReissueWriter}
                >
                  {props.writerReissuing ? "再発行中…" : "Writer clientを再発行"}
                </Button>
                {props.writerStatus && (
                  <p className="mt-3 text-xs leading-prose text-text-secondary">
                    {props.writerStatus}
                  </p>
                )}
              </Panel>
            )}

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

function cliTokenStatus(expiresAt: string) {
  const remaining = new Date(expiresAt).getTime() - Date.now();
  if (remaining <= 0) {
    return { state: "expired", label: "期限切れ", detail: "再発行が必要" };
  }
  if (remaining <= expiringSoonMilliseconds) {
    return {
      state: "pending",
      label: "期限間近",
      detail: `あと${Math.ceil(remaining / (24 * 60 * 60 * 1000))}日`,
    };
  }
  return { state: "active", label: "有効", detail: "" };
}

function roleLabel(role: string) {
  if (role === "owner") return "オーナー";
  if (role === "editor") return "編集者";
  if (role === "reader") return "閲覧者";
  return "公開";
}
