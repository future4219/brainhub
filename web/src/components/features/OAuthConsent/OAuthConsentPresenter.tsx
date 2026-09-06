import { Link } from "react-router-dom";

import { AppShell } from "@/components/ui/AppShell";
import { Button, buttonVariants } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { Panel } from "@/components/ui/Panel";
import { loginUrl } from "@/config/url";
import type { OAuthConsent } from "@/entities/mcp/entity";
import type { ViewerState } from "@/entities/user/entity";

type OAuthConsentPresenterProps = ViewerState & {
  consent: OAuthConsent | null;
  error: string;
  submitting: boolean;
  loginNext: string;
  onApprove: () => void;
  onDeny: () => void;
};

export function OAuthConsentPresenter({
  consent,
  error,
  submitting,
  loginNext,
  onApprove,
  onDeny,
  ...shell
}: OAuthConsentPresenterProps) {
  return (
    <AppShell {...shell} crumbs={["接続を許可"]}>
      <main className="mx-auto w-full max-w-copy px-4 pb-20 pt-8 sm:pt-12">
        <header>
          <p className="font-mono text-xs uppercase tracking-section text-text-muted">
            oauth authorization
          </p>
          <h1 className="mt-2 text-title font-semibold tracking-tight">
            Brainhubへの接続を許可しますか
          </h1>
        </header>

        <Panel className="mt-8">
          {shell.viewer === undefined && (
            <Feedback kind="loading">sessionを確認中…</Feedback>
          )}
          {shell.viewer === null && (
            <div className="p-4">
              <p className="text-body text-text-secondary">
                認可を続けるにはbrainhubへログインしてください。
              </p>
              <Link
                className={`${buttonVariants({ size: "sm" })} mt-4`}
                to={loginUrl(loginNext)}
              >
                ログイン
              </Link>
            </div>
          )}
          {shell.viewer && !consent && !error && (
            <Feedback kind="loading">requestを確認中…</Feedback>
          )}
          {error && <Feedback kind="error">{error}</Feedback>}
          {shell.viewer && consent && (
            <div className="p-4 sm:p-6">
              <dl className="space-y-4 text-ui">
                <div>
                  <dt className="text-text-muted">クライアント</dt>
                  <dd className="mt-1 break-words text-section font-semibold text-text">
                    {consent.client_name === "codex"
                      ? "Codex"
                      : consent.client_name}
                  </dd>
                </div>
                <div>
                  <dt className="text-text-muted">許可する内容</dt>
                  <dd className="mt-1 text-text-secondary">
                    あなたが見られる脳の検索とページの読み取り。
                    脳への参加・退出や公開範囲の変更は、この接続にも反映されます。
                  </dd>
                </div>
                <div>
                  <dt className="text-text-muted">書き込み</dt>
                  <dd className="mt-1 text-text-secondary">
                    この接続は読み取り専用です。ページの作成・変更はできません。
                  </dd>
                </div>
              </dl>
              <h2 className="mt-6 text-body font-semibold">現在読める脳</h2>
              {consent.visible_brains.length === 0 ? (
                <p className="mt-2 text-ui text-text-muted">
                  現在読める脳はありません。脳を作るか招待を受けると、この接続から利用できます。
                </p>
              ) : (
                <ul className="mt-2 divide-y divide-divider">
                  {consent.visible_brains.map((brain) => (
                    <li
                      key={brain.source_id}
                      className="flex items-center justify-between gap-4 py-3 text-ui"
                    >
                      <div className="min-w-0">
                        <p className="break-words font-semibold">{brain.name}</p>
                        <p className="mt-1 break-all font-mono text-xs text-text-muted">
                          {brain.source_id}
                        </p>
                      </div>
                      <span className="shrink-0 rounded-badge border border-border-control px-2 py-1 text-xs text-text-secondary">
                        読み取り
                      </span>
                    </li>
                  ))}
                </ul>
              )}
              <p className="mt-4 border-t border-border pt-4 text-ui text-text-secondary">
                許可すると接続元のAIに戻ります。
              </p>
              <div className="mt-4 flex flex-wrap gap-2 sm:justify-end">
                <Button type="button" disabled={submitting} onClick={onApprove}>
                  {submitting ? "処理中…" : "許可する"}
                </Button>
                <Button
                  variant="outline"
                  type="button"
                  disabled={submitting}
                  onClick={onDeny}
                >
                  許可しない
                </Button>
              </div>
            </div>
          )}
        </Panel>
      </main>
    </AppShell>
  );
}
