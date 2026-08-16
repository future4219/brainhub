import { Link } from "react-router-dom";

import { AppShell } from "@/components/ui/AppShell";
import { buttonVariants } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { Panel } from "@/components/ui/Panel";
import { appUrl } from "@/config/url";
import type { ViewerState } from "@/entities/user/entity";

type StatusPresenterProps = ViewerState & { label: string; title: string };

export function StatusPresenter({
  label,
  title,
  ...shell
}: StatusPresenterProps) {
  return (
    <AppShell {...shell} crumbs={[label]}>
      <main className="mx-auto max-w-content px-4 pb-20 pt-8 sm:px-8">
        <div className="mx-auto max-w-credentials">
          <Panel>
            <header className="border-b border-border p-4">
              <p className="font-mono text-xs uppercase tracking-section text-text-muted">
                {label}
              </p>
              <h1 className="mt-2 text-title font-semibold">{title}</h1>
            </header>
            <Feedback kind="empty">
              URLを確認するか、脳の一覧へ戻ってください。
            </Feedback>
            <div className="p-4">
              <Link
                className={buttonVariants({ variant: "outline" })}
                to={appUrl.brainList}
              >
                脳の一覧へ戻る
              </Link>
            </div>
          </Panel>
        </div>
      </main>
    </AppShell>
  );
}
