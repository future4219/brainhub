import { Link } from "react-router-dom";

import type { PagesSectionProps } from "@/components/features/BrainDetail/types";
import { Feedback } from "@/components/ui/Feedback";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import { pageUrl } from "@/config/url";
import { formatDate } from "@/lib/format";

export function PagesSection(props: PagesSectionProps) {
  if (props.error) {
    return (
      <Feedback kind="error" className="mt-6">
        {props.error}
      </Feedback>
    );
  }
  if (props.pages === null) {
    return (
      <Feedback kind="loading" className="mt-6">
        loading pages…
      </Feedback>
    );
  }
  return (
    <div className="mt-6 flex flex-col items-start gap-6 lg:flex-row">
      <aside
        className="w-full shrink-0 lg:sticky lg:top-sticky lg:w-type-tree"
        aria-label="ページtypeで絞り込む"
      >
        <Panel>
          <div className="border-b border-border px-4 py-2 font-mono text-xs uppercase tracking-label text-text-muted">
            signature
          </div>
          <div className="py-2 font-mono text-sm">
            <TypeRow
              label={`${props.sourceID}/`}
              count={props.pages.length}
              selected={props.selectedType === "all"}
              onClick={() => props.onType("all")}
            />
            {props.typeCounts.map(([type, count], index) => (
              <TypeRow
                label={`${index === props.typeCounts.length - 1 ? "└──" : "├──"} ${type}`}
                count={count}
                selected={props.selectedType === type}
                onClick={() => props.onType(type)}
                key={type}
              />
            ))}
          </div>
        </Panel>
        <p className="mt-2 px-1 text-xs text-text-muted">
          選択すると一覧を絞り込む。選択行は全体を反転する。
        </p>
      </aside>

      <section className="min-w-0 flex-1" aria-labelledby="page-index-heading">
        <div className="mb-4 sm:hidden">
          <Input
            aria-label="ページを検索"
            value={props.query}
            placeholder="ページを検索"
            onChange={(event) => props.onQuery(event.target.value)}
          />
        </div>
        <Panel>
          <div className="flex items-center gap-3 border-b border-border px-4 py-2">
            <h2
              id="page-index-heading"
              className="font-mono text-sm text-text-secondary"
            >
              {props.visiblePages.length} pages
              {props.selectedType !== "all"
                ? ` · type = ${props.selectedType}`
                : ""}
            </h2>
            <div className="flex-1" />
            {(["updated", "slug"] as const).map((value) => (
              <button
                className={
                  props.sort === value
                    ? "h-control-sm rounded-control border border-border-strong bg-surface px-3 font-mono text-xs"
                    : "h-control-sm border border-transparent px-3 font-mono text-xs text-text-muted"
                }
                type="button"
                aria-pressed={props.sort === value}
                onClick={() => props.onSort(value)}
                key={value}
              >
                {value}
              </button>
            ))}
          </div>
          {props.pages.length === 0 && (
            <Feedback kind="empty">
              公開ページがありません。ページが追加されるとここに索引が表示されます。
            </Feedback>
          )}
          {props.pages.length > 0 && props.visiblePages.length === 0 && (
            <Feedback kind="empty">
              一致するページがありません。typeまたは検索語を変更してください。
            </Feedback>
          )}
          {props.visiblePages.map((page) => (
            <Link
              className="page-row-grid grid gap-3 border-b border-divider px-4 py-3 last:border-b-0 hover:bg-surface"
              to={pageUrl(props.sourceID, page.slug)}
              key={page.slug}
            >
              <div className="min-w-0">
                <h3 className="text-body leading-row">{page.title}</h3>
                <p className="mt-1 break-words font-mono text-xs text-text-muted">
                  {page.slug}
                </p>
              </div>
              <span className="rounded-badge border border-border-control px-2 py-0.5 font-mono text-xs text-text-tertiary">
                {page.type}
              </span>
              <time
                className="text-right font-mono text-xs text-text-muted"
                dateTime={page.updated_at}
              >
                {formatDate(page.updated_at)}
              </time>
            </Link>
          ))}
          {props.pages.length > 0 && (
            <footer className="flex items-center justify-between gap-4 px-4 py-3 font-mono text-xs text-text-muted">
              <span>行を選ぶと本文を表示</span>
              <span>
                {props.visiblePages.length} / {props.pages.length}
              </span>
            </footer>
          )}
        </Panel>
      </section>
    </div>
  );
}

function TypeRow({
  label,
  count,
  selected,
  onClick,
}: {
  label: string;
  count: number;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      className={
        selected
          ? "type-row-grid grid w-full items-center gap-3 bg-text px-3 py-2 text-left text-canvas"
          : "type-row-grid grid w-full items-center gap-3 px-3 py-2 text-left text-text-tertiary hover:bg-surface hover:text-text"
      }
      type="button"
      aria-pressed={selected}
      onClick={onClick}
    >
      <span className="truncate">{label}</span>
      <span className="text-right text-xs tabular-nums">{count}</span>
    </button>
  );
}
