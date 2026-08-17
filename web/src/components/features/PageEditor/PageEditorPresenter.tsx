import type { FormEvent, KeyboardEvent } from "react";

import { BrainDetailLayout } from "@/components/features/BrainDetail/BrainDetailLayout";
import { Button } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import type { Brain, Page, PageType } from "@/entities/brain/entity";
import type { ViewerState } from "@/entities/user/entity";

type Props = {
  shell: ViewerState;
  sourceID: string;
  brain: Brain | null;
  editing: boolean;
  error: "forbidden" | "not-found" | "load" | "";
  pages: Page[];
  pageTypes: PageType[];
  directories: string[];
  slug: string;
  title: string;
  type: string;
  tags: string[];
  tagInput: string;
  supersededBy: string;
  compiledTruth: string;
  existingTimeline: string;
  timelineEntry: string;
  showTimelineEntry: boolean;
  saving: boolean;
  saveError: string;
  onSlug: (value: string) => void;
  onTitle: (value: string) => void;
  onType: (value: string) => void;
  onTagInput: (value: string) => void;
  onAddTag: () => void;
  onRemoveTag: (value: string) => void;
  onSupersededBy: (value: string) => void;
  onCompiledTruth: (value: string) => void;
  onTimelineEntry: (value: string) => void;
  onShowTimelineEntry: () => void;
  onCancel: () => void;
  onSave: () => void;
};

export function PageEditorPresenter(props: Props) {
  function submit(event: FormEvent) {
    event.preventDefault();
    props.onSave();
  }

  function tagKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      props.onAddTag();
    }
  }

  return (
    <BrainDetailLayout
      shell={props.shell}
      detail={{
        sourceID: props.sourceID,
        tab: "pages",
        brain: props.brain,
        error: props.error === "not-found" ? "not-found" : props.error === "load" ? "load" : "",
      }}
    >
      {props.error === "forbidden" && (
        <Feedback kind="empty" className="mt-6">
          この画面は owner または editor のみ利用できます。
        </Feedback>
      )}
      {!props.error && !props.brain && (
        <Feedback kind="loading" className="mt-6">loading editor…</Feedback>
      )}
      {!props.error && props.brain && (
        <form className="mt-6" onSubmit={submit}>
          <Panel className="overflow-hidden">
            <div className="border-b border-border px-4 py-4">
              <h2 className="text-section font-semibold">
                {props.editing ? "ページを編集" : "ページを作成"}
              </h2>
              <p className="mt-2 text-ui text-text-secondary">
                Compiled Truth は上書き、Timeline は新しい1行だけを追記します。
              </p>
            </div>
            <div className="px-4">
              <FormField
                label="slug"
                hint={props.editing ? "slug は不変です。変更する場合は新しいページを作り、古いページに superseded_by を設定してください。" : "英小文字・数字・ハイフン・スラッシュのみ。既存ディレクトリは候補から選べます。"}
              >
                <Input
                  className="font-mono"
                  value={props.slug}
                  readOnly={props.editing}
                  required
                  pattern="[a-z0-9/-]+"
                  list={props.editing ? undefined : "page-directories"}
                  onChange={(event) => props.onSlug(event.target.value)}
                />
                {!props.editing && (
                  <datalist id="page-directories">
                    {props.directories.map((directory) => <option value={directory} key={directory} />)}
                  </datalist>
                )}
              </FormField>
              <FormField label="title">
                <Input value={props.title} required onChange={(event) => props.onTitle(event.target.value)} />
              </FormField>
              <FormField label="type" hint="現在の GBrain スキーマパックが宣言した型だけを選べます。">
                <select className="field-control" value={props.type} required onChange={(event) => props.onType(event.target.value)}>
                  {props.pageTypes.map((pageType) => (
                    <option value={pageType.name} key={pageType.name}>{pageType.name}</option>
                  ))}
                </select>
              </FormField>
              <FormField label="tags" hint="Enter またはカンマで追加します。">
                <div className="flex flex-wrap gap-2">
                  {props.tags.map((tag) => (
                    <button className="rounded-pill border border-border-control px-3 py-1 font-mono text-xs" type="button" aria-label={`${tag} を削除`} onClick={() => props.onRemoveTag(tag)} key={tag}>
                      {tag} ×
                    </button>
                  ))}
                </div>
                <div className="flex gap-2">
                  <Input value={props.tagInput} placeholder="product" onChange={(event) => props.onTagInput(event.target.value)} onKeyDown={tagKeyDown} />
                  <Button variant="outline" type="button" onClick={props.onAddTag}>追加</Button>
                </div>
              </FormField>
              <FormField label="superseded_by" hint="状態は本文の Status ではなく、明示的なグラフ辺から解決されます。">
                <select className="field-control font-mono" value={props.supersededBy} onChange={(event) => props.onSupersededBy(event.target.value)}>
                  <option value="">なし</option>
                  {props.pages.filter((page) => page.slug !== props.slug).map((page) => (
                    <option value={page.slug} key={page.slug}>{page.slug}</option>
                  ))}
                </select>
              </FormField>
            </div>
          </Panel>

          <Panel className="mt-6 overflow-hidden">
            <div className="border-b border-border px-4 py-3">
              <h3 className="font-mono text-xs uppercase tracking-label">Compiled Truth</h3>
              <p className="mt-1 text-xs text-text-muted">保存すると現在の内容で上書きされます。</p>
            </div>
            <textarea className="field-control min-h-80 rounded-none border-0 font-mono leading-prose" value={props.compiledTruth} aria-label="Compiled Truth" onChange={(event) => props.onCompiledTruth(event.target.value)} />
          </Panel>

          <Panel className="mt-6 overflow-hidden">
            <div className="flex items-center justify-between gap-4 border-b border-border px-4 py-3">
              <div>
                <h3 className="font-mono text-xs uppercase tracking-label">Timeline</h3>
                <p className="mt-1 text-xs text-text-muted">既存エントリは読み取り専用です。</p>
              </div>
              {!props.showTimelineEntry && (
                <Button variant="outline" size="sm" type="button" onClick={props.onShowTimelineEntry}>＋ エントリを追加</Button>
              )}
            </div>
            {props.existingTimeline ? (
              <pre className="whitespace-pre-wrap border-b border-divider p-4 font-mono text-sm leading-prose text-text-secondary">{props.existingTimeline}</pre>
            ) : (
              <Feedback kind="empty">既存エントリはありません。</Feedback>
            )}
            {props.showTimelineEntry && (
              <div className="p-4">
                <Input value={props.timelineEntry} required placeholder="2026-08-17 変更内容" aria-label="追加する Timeline エントリ" onChange={(event) => props.onTimelineEntry(event.target.value)} />
              </div>
            )}
          </Panel>

          <Feedback kind="info" className="mt-6">
            同時編集の検出機能はありません。CLI や接続された AI からも更新されるため、保存前に最新内容を確認してください。
          </Feedback>
          {props.saveError && <Feedback kind="error" className="mt-4">{props.saveError}</Feedback>}
          <div className="mt-6 flex justify-end gap-3">
            <Button variant="outline" type="button" disabled={props.saving} onClick={props.onCancel}>キャンセル</Button>
            <Button type="submit" disabled={props.saving || props.pageTypes.length === 0}>{props.saving ? "保存中…" : "保存"}</Button>
          </div>
        </form>
      )}
    </BrainDetailLayout>
  );
}
