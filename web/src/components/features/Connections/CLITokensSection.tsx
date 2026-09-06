import type { ConnectSectionProps } from "@/components/features/Connections/types";
import type { MCPCLIToken } from "@/entities/mcp/entity";
import { Button } from "@/components/ui/Button";
import { CopyValue } from "@/components/ui/CopyValue";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { copyLabel, formatDate } from "@/lib/format";

type CLITokensSectionProps = Pick<
  ConnectSectionProps,
  | "cliTokenLabel"
  | "createdCLIToken"
  | "cliTokenSubmitting"
  | "cliTokenRevoking"
  | "cliTokenError"
  | "copyState"
  | "onCopy"
  | "onIssueCLIToken"
  | "onCLITokenLabel"
  | "onRevokeCLIToken"
> & { mcpURL: string; tokens: MCPCLIToken[] };

const expiringSoonMilliseconds = 30 * 24 * 60 * 60 * 1000;

export function CLITokensSection(props: CLITokensSectionProps) {
  const cliConfig = `[mcp_servers.brainhub]\nurl = "${props.mcpURL}"\nbearer_token_env_var = "BRAINHUB_MCP_TOKEN"`;
  return (
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
              onChange={(event) => props.onCLITokenLabel(event.target.value)}
            />
          </FormField>
          <Button
            type="submit"
            disabled={props.cliTokenSubmitting || !props.cliTokenLabel.trim()}
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
        {props.tokens.length === 0 ? (
          <Feedback kind="empty">
            CLI用トークンはまだありません。上のフォームから、使う端末ごとに発行してください。
          </Feedback>
        ) : (
          <div>
            {props.tokens.map((token) => {
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
                    {props.cliTokenRevoking === token.id ? "失効中…" : "失効"}
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
