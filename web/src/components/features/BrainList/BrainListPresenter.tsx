import { Link } from "react-router-dom";

import type {
  BrainMetrics,
  Visibility,
} from "@/components/features/BrainList/types";
import { AppShell } from "@/components/ui/AppShell";
import { Feedback } from "@/components/ui/Feedback";
import { Panel } from "@/components/ui/Panel";
import { StateBadge } from "@/components/ui/StateBadge";
import { brainUrl } from "@/config/url";
import type { Brain } from "@/entities/brain/entity";
import type { ViewerState } from "@/entities/user/entity";
import { formatDate, pageSignature } from "@/lib/format";

type BrainListPresenterProps = ViewerState & {
  brains: Brain[] | null;
  visibleBrains: Brain[];
  visibility: Visibility;
  metrics: Record<string, BrainMetrics>;
  query: string;
  error: boolean;
  onVisibility: (visibility: Visibility) => void;
  onQuery: (query: string) => void;
};

export function BrainListPresenter({
  brains,
  visibleBrains,
  visibility,
  metrics,
  query,
  error,
  onVisibility,
  onQuery,
  ...shell
}: BrainListPresenterProps) {
  return (
    <AppShell
      {...shell}
      crumbs={["脳"]}
      brains={brains}
      search={{ value: query, placeholder: "脳を検索", onChange: onQuery }}
    >
      <main className="w-full px-4 pb-20 pt-8 sm:px-8">
        <header className="mb-6 flex items-start justify-between gap-6">
          <div>
            <h1 className="text-title font-semibold tracking-tight">脳</h1>
            <p className="mt-2 max-w-copy text-body leading-copy text-text-secondary">
              領域ごとの知識ベース。接続情報を AI
              クライアントに渡すと、そのクライアントが中身を読めるようになる。
            </p>
          </div>
        </header>

        <Panel>
          <div className="flex items-center gap-2 border-b border-border px-4 py-2">
            {(["all", "public", "private"] as const).map((value) => (
              <button
                className={
                  visibility === value
                    ? "h-control-sm rounded-control border border-border-strong bg-surface-selected px-3 text-sm"
                    : "h-control-sm rounded-control border border-transparent px-3 text-sm text-text-muted hover:text-text"
                }
                type="button"
                aria-pressed={visibility === value}
                onClick={() => onVisibility(value)}
                key={value}
              >
                {value === "all" ? "すべて" : value}
              </button>
            ))}
            <div className="flex-1" />
            {brains && (
              <span className="font-mono text-xs text-text-muted">
                {visibleBrains.length} brains
              </span>
            )}
          </div>

          {error && (
            <Feedback kind="error">
              脳の一覧を取得できません。接続を確認して再読み込みしてください。
            </Feedback>
          )}
          {!error && brains === null && (
            <Feedback kind="loading">loading brains…</Feedback>
          )}
          {!error && brains?.length === 0 && (
            <Feedback kind="empty">
              まだ脳がありません。ログイン後、「新しい脳」から最初の領域を作成できます。
            </Feedback>
          )}
          {!error &&
            brains &&
            brains.length > 0 &&
            visibleBrains.length === 0 && (
              <Feedback kind="empty">
                この公開範囲の脳はありません。別のフィルターを選んでください。
              </Feedback>
            )}
          {visibleBrains.length > 0 && (
            <div className="overflow-x-auto">
              <div
                className="brain-table-grid grid min-w-credentials gap-4 border-b border-border bg-table-head px-4 py-2 font-mono text-xs uppercase tracking-label text-text-muted"
                aria-hidden="true"
              >
                <span>name</span>
                <span>signature</span>
                <span className="text-right">pages</span>
                <span>updated</span>
                <span>state</span>
              </div>
              {visibleBrains.map((brain) => (
                <BrainRow
                  brain={brain}
                  metrics={metrics[brain.source_id]}
                  key={brain.id}
                />
              ))}
            </div>
          )}
        </Panel>
      </main>
    </AppShell>
  );
}

function BrainRow({
  brain,
  metrics,
}: {
  brain: Brain;
  metrics?: BrainMetrics;
}) {
  const pages = metrics?.pages;
  const content = (
    <>
      <div className="min-w-0">
        <strong className="block truncate font-mono text-body font-medium">
          {brain.source_id}
        </strong>
        <span className="mt-1 block truncate font-mono text-xs text-text-muted">
          {brain.visibility}
        </span>
      </div>
      <span className="truncate font-mono text-xs text-text-muted">
        {pages ? pageSignature(pages.map((page) => page.type)) : "—"}
      </span>
      <span className="text-right font-mono text-ui tabular-nums">
        {pages?.length ?? "—"}
      </span>
      <time
        className="font-mono text-xs text-text-muted"
        dateTime={brain.updated_at}
      >
        {formatDate(brain.updated_at)}
      </time>
      <StateBadge state={brain.state} />
      {brain.state_reason && (
        <p className="col-span-full text-xs text-text-secondary">
          {brain.state_reason}
        </p>
      )}
    </>
  );
  const className =
    "brain-table-grid grid min-w-credentials items-center gap-4 border-b border-divider px-4 py-3 last:border-b-0 hover:bg-surface";

  return brain.state === "ready" ? (
    <Link className={className} to={brainUrl(brain.source_id)}>
      {content}
    </Link>
  ) : (
    <div className={`${className} text-text-muted`} aria-disabled="true">
      {content}
    </div>
  );
}
