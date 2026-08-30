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
      <main className="mx-auto max-w-credentials px-4 pb-20 pt-12 sm:px-8">
        <header>
          <p className="font-mono text-xs uppercase tracking-section text-text-muted">
            oauth authorization
          </p>
          <h1 className="mt-2 text-title font-semibold tracking-tight">
            このクライアントに脳へのアクセスを許可しますか
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
            <div className="p-4">
              <dl className="space-y-4 text-ui">
                <div>
                  <dt className="text-text-muted">クライアント</dt>
                  <dd className="mt-1 font-mono text-text">
                    {consent.client_name}
                  </dd>
                </div>
                <div>
                  <dt className="text-text-muted">許可する内容</dt>
                  <dd className="mt-1 text-text-secondary">
                    現在あなたが見られる脳の読み取り。Membershipや公開範囲の変更は次の呼び出しから反映されます。
                  </dd>
                </div>
                <div>
                  <dt className="text-text-muted">書き込み</dt>
                  <dd className="mt-1 text-text">許可しない</dd>
                </div>
              </dl>
              <div className="mt-6 flex gap-2">
                <Button
                  type="button"
                  disabled={submitting}
                  onClick={onApprove}
                >
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
