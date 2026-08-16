import { Link } from "react-router-dom";

import { AppShell } from "@/components/ui/AppShell";
import { Button, buttonVariants } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { Panel } from "@/components/ui/Panel";
import { invitationUrl, loginUrl, registerUrl } from "@/config/url";
import type { InvitationPreview } from "@/entities/access/entity";
import type { ViewerState } from "@/entities/user/entity";

type InvitationPresenterProps = ViewerState & {
  token: string;
  preview: InvitationPreview | null;
  notFound: boolean;
  error: string;
  submitting: boolean;
  onAccept: () => void;
};

export function InvitationPresenter({
  token,
  preview,
  notFound,
  error,
  submitting,
  onAccept,
  ...shell
}: InvitationPresenterProps) {
  const next = invitationUrl(token);
  return (
    <AppShell {...shell} crumbs={["招待"]}>
      <main className="mx-auto max-w-content px-4 pb-20 pt-8 sm:px-8">
        <div className="mx-auto max-w-credentials">
          <header className="mb-6">
            <p className="font-mono text-xs uppercase tracking-section text-text-muted">
              repository invitation
            </p>
            <h1 className="mt-2 text-title font-semibold tracking-tight">
              {preview?.brain_name ?? "招待"}
            </h1>
          </header>
          <Panel>
            {notFound && (
              <Feedback kind="empty">
                この招待は見つからないか、期限切れです。新しい招待リンクを発行者へ依頼してください。
              </Feedback>
            )}
            {!notFound && !preview && !error && (
              <Feedback kind="loading">loading invitation…</Feedback>
            )}
            {error && <Feedback kind="error">{error}</Feedback>}
            {preview && (
              <div className="p-4">
                <p className="text-body text-text-secondary">
                  <strong className="text-text">
                    {preview.invited_by_name}
                  </strong>{" "}
                  から脳への招待が届いています。
                </p>
                {shell.viewer === undefined && (
                  <Feedback kind="loading" className="mt-4">
                    sessionを確認中…
                  </Feedback>
                )}
                {shell.viewer === null && (
                  <div className="mt-6">
                    <p className="mb-4 text-ui text-text-secondary">
                      受諾するにはログイン、または新規登録してください。完了後にこの招待へ戻ります。
                    </p>
                    <div className="flex flex-wrap gap-3">
                      <Link className={buttonVariants()} to={loginUrl(next)}>
                        ログイン
                      </Link>
                      <Link
                        className={buttonVariants({ variant: "outline" })}
                        to={registerUrl(next)}
                      >
                        新規登録
                      </Link>
                    </div>
                  </div>
                )}
                {shell.viewer && (
                  <Button
                    className="mt-6"
                    type="button"
                    disabled={submitting}
                    onClick={onAccept}
                  >
                    {submitting ? "受諾中…" : "招待を受諾する"}
                  </Button>
                )}
              </div>
            )}
          </Panel>
        </div>
      </main>
    </AppShell>
  );
}
