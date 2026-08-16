import { Link } from "react-router-dom";

import { BrainDetailLayout } from "@/components/features/BrainDetail/BrainDetailLayout";
import { buttonVariants } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { Panel } from "@/components/ui/Panel";
import { brainUrl } from "@/config/url";
import type { Brain, PageDetail } from "@/entities/brain/entity";
import type { ViewerState } from "@/entities/user/entity";
import { formatDate } from "@/lib/format";

type BrainPagePresenterProps = {
  shell: ViewerState;
  sourceID: string;
  brain: Brain | null;
  brainError: "not-found" | "load" | "";
  page: PageDetail | null;
  pageError: "not-found" | "load" | "";
};

export function BrainPagePresenter({
  shell,
  sourceID,
  brain,
  brainError,
  page,
  pageError,
}: BrainPagePresenterProps) {
  return (
    <BrainDetailLayout
      shell={shell}
      detail={{ sourceID, tab: "pages", brain, error: brainError }}
    >
      {pageError && (
        <Feedback kind={pageError === "not-found" ? "empty" : "error"} className="mt-6">
          {pageError === "not-found"
            ? "このページは見つかりません。ページ一覧へ戻ってください。"
            : "ページを読み込めません。接続を確認して再読み込みしてください。"}
        </Feedback>
      )}
      {!pageError && !page && (
        <Feedback kind="loading" className="mt-6">
          loading page…
        </Feedback>
      )}
      {page && (
        <article className="mt-6">
          <Link
            className={buttonVariants({ variant: "outline", size: "sm" })}
            to={brainUrl(sourceID)}
          >
            ← ページ一覧
          </Link>
          <header className="mt-6 border-b border-border pb-6">
            <div className="flex flex-wrap items-center gap-3 font-mono text-xs text-text-muted">
              <span>{page.type}</span>
              <span>{page.slug}</span>
              <time dateTime={page.updated_at}>{formatDate(page.updated_at)}</time>
            </div>
            <h2 className="mt-3 max-w-content text-title font-semibold leading-copy">
              {page.title}
            </h2>
          </header>
          <Panel className="mt-6 overflow-hidden">
            {page.compiled_truth ? (
              <pre className="whitespace-pre-wrap break-words p-6 font-sans text-body leading-prose text-text-secondary">
                {page.compiled_truth}
              </pre>
            ) : (
              <Feedback kind="empty">本文がありません。</Feedback>
            )}
          </Panel>
          {page.timeline && (
            <Panel className="mt-6 overflow-hidden">
              <h3 className="border-b border-border px-6 py-3 font-mono text-xs uppercase tracking-label text-text-muted">
                timeline
              </h3>
              <pre className="whitespace-pre-wrap break-words p-6 font-sans text-body leading-prose text-text-secondary">
                {page.timeline}
              </pre>
            </Panel>
          )}
        </article>
      )}
    </BrainDetailLayout>
  );
}
