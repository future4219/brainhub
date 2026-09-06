import { Link } from "react-router-dom";

import type { ConnectSectionProps } from "@/components/features/Connections/types";
import { copyLabel } from "@/lib/format";
import { Button, buttonVariants } from "@/components/ui/Button";
import { CopyValue } from "@/components/ui/CopyValue";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { StateBadge } from "@/components/ui/StateBadge";
import { appUrl, loginUrl } from "@/config/url";
import { codexConnectCommand, connectionSteps, formatDate } from "@/lib/format";

const expiringSoonMilliseconds = 30 * 24 * 60 * 60 * 1000;

export function ConnectSection(props: ConnectSectionProps) {
  const codexClient = props.connection?.codex_client;
  const codexCommand =
    props.config && codexClient
      ? codexConnectCommand(props.config.mcp_url, codexClient.id)
      : "";
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
        <h1 className="text-title font-semibold">AIとの接続</h1>
        <p className="mt-2 max-w-copy text-body leading-copy text-text-secondary">
          BrainhubをAIに追加すると、脳の検索・閲覧と、編集権限のある脳への保存ができます。
          脳が増えても繋ぎ直す必要はありません。
        </p>
      </header>

      {props.error && <Feedback kind="error">{props.error}</Feedback>}
      {props.viewer === undefined && (
        <Feedback kind="loading">ログイン状態を確認中…</Feedback>
      )}
      {!props.config && !props.error && (
        <Feedback kind="loading">接続情報を読み込み中…</Feedback>
      )}
      {props.config && props.viewer === null && (
        <Panel className="p-4">
          <p className="text-ui text-text-secondary">
            接続情報を作るにはログインしてください。
          </p>
          <Link
            className={`${buttonVariants({ size: "sm" })} mt-4`}
            to={loginUrl(appUrl.connections)}
          >
            ログイン
          </Link>
        </Panel>
      )}
      {props.config && props.viewer && !props.connection && !props.error && (
        <Feedback kind="loading">接続情報を読み込み中…</Feedback>
      )}
      {props.config && props.viewer && props.connection && (
        <div className="max-w-copy space-y-6">
          {props.connection.reader?.state === "orphan" && (
            <Panel className="p-4">
              <StateBadge state="orphan" />
              <h3 className="mt-3 text-body font-semibold">
                読み取り接続を再発行してください
              </h3>
              <p className="mt-2 text-ui leading-prose text-text-secondary">
                GBrainのreader作成または範囲更新に失敗したため、現在のリクエストは転送されません。
                再発行後にAIからもう一度呼び出してください。
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
            <SectionHeading eyebrow="codex" title="CodexにBrainhubを追加" />
            <div className="p-4">
              <ol className="list-decimal space-y-3 pl-5 text-body text-text-secondary">
                <li>下のコマンドをCodexを使う端末で実行します。</li>
                <li>
                  開いたブラウザでBrainhubにログインし、読み取り・書き込みの権限を確認して許可します。
                </li>
                <li>
                  端末に接続完了が表示されたら、Codexで新しい会話を開いて使えます。
                </li>
              </ol>
              {codexClient ? (
                <CopyValue
                  label="Codexへ追加するコマンド"
                  value={codexCommand}
                  copyLabel={copyLabel(props.copyState["codex-command"])}
                  onCopy={() => props.onCopy("codex-command", codexCommand)}
                />
              ) : (
                <Button
                  className="mt-4"
                  type="button"
                  disabled={props.submitting}
                  onClick={() => props.onIssue("codex")}
                >
                  {props.submitting ? "準備中…" : "Codexの接続を準備"}
                </Button>
              )}
              <p className="mt-3 text-ui text-text-secondary">
                接続後は「Brainhubで〇〇について調べて」「〇〇の脳にこの内容を保存して」と話しかけてください。
              </p>
              <details className="mt-4 text-ui text-text-secondary">
                <summary className="cursor-pointer">
                  ブラウザが開かない・認証をやり直す
                </summary>
                <p className="mt-2">
                  端末に表示された認証URLをブラウザで開いてください。やり直す場合は同じ端末で次を実行します。
                </p>
                <code className="mt-2 block">
                  codex mcp login brainhub --scopes read,write
                </code>
                <p className="mt-2">
                  以前の接続は読み取り専用のままです。書き込みを使うには、再認可で「読み取り・書き込み」を選んで新しい会話を開いてください。
                </p>
              </details>
              {props.copyState["codex-command"] === "failed" && (
                <Feedback kind="error">
                  コピーできません。コマンドを選択してコピーしてください。
                </Feedback>
              )}
            </div>
          </Panel>

          <Panel>
            <SectionHeading eyebrow="access" title="見られる脳" />
            {props.connection.visible_brains.length === 0 ? (
              <Feedback kind="empty">
                現在読める脳はありません。脳を作るか、招待を受けると利用できます。
              </Feedback>
            ) : (
              props.connection.visible_brains.map((brain) => (
                <div
                  className="flex items-center justify-between gap-4 border-b border-divider p-4 last:border-b-0"
                  key={brain.source_id}
                >
                  <div className="min-w-0">
                    <p className="truncate text-body text-text">{brain.name}</p>
                    <p className="mt-1 text-xs text-text-muted">
                      {brain.source_id} · {roleLabel(brain.role)} ·{" "}
                      {brain.can_write
                        ? "書き込み許可時に編集可能"
                        : "読み取り"}
                    </p>
                  </div>
                  <StateBadge state={brain.state} />
                </div>
              ))
            )}
            <p className="border-t border-divider p-4 text-ui text-text-secondary">
              脳への参加・退出や公開範囲の変更は、次の呼び出しから反映されます。
            </p>
          </Panel>

          <details>
            <summary className="cursor-pointer py-3 text-body font-semibold">
              手動トークンで接続する
            </summary>
            <Panel>
              <SectionHeading eyebrow="cli" title="CLI用トークン" />
              <div className="border-b border-divider p-4 text-ui leading-prose text-text-secondary">
                ブラウザ認証を使えない環境向けの読み取り専用トークンです。有効期限は90日で、自動更新はされません。
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
                            発行 {formatDate(token.created_at)} · 期限{" "}
                            {formatDate(token.expires_at)}
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
          </details>
          <details>
            <summary className="cursor-pointer py-3 text-body font-semibold">
              Claude Webに接続する
            </summary>
            <Panel>
              <SectionHeading eyebrow="claude web" title="接続手順" />
              <div className="px-4">
                <CopyValue
                  label="MCP URL"
                  value={props.config.mcp_url}
                  copyLabel={copyLabel(props.copyState.mcp)}
                  onCopy={() => props.onCopy("mcp", props.config!.mcp_url)}
                />
                {clientID ? (
                  <CopyValue
                    label="OAuth Client ID"
                    value={clientID}
                    copyLabel={copyLabel(props.copyState.client)}
                    onCopy={() => props.onCopy("client", clientID)}
                  />
                ) : (
                  <Button
                    className="my-4"
                    type="button"
                    disabled={props.submitting}
                    onClick={() => props.onIssue("claude-web")}
                  >
                    Claude Webの接続を準備
                  </Button>
                )}
                {(props.copyState.mcp === "failed" ||
                  props.copyState.client === "failed") && (
                  <Feedback kind="error">
                    コピーできません。値を選択してコピーしてください。
                  </Feedback>
                )}
              </div>
              <div className="border-b border-divider p-4 text-ui leading-prose text-text-secondary">
                Dynamic Client
                Registrationには対応していません。最初の失敗後にClient
                IDを手入力します。
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
          </details>
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
