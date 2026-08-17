import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { PageEditorPresenter } from "@/components/features/PageEditor/PageEditorPresenter";
import { brainUrl, pageUrl } from "@/config/url";
import type { Brain, Page, PageDetail, PageType } from "@/entities/brain/entity";
import { useViewer } from "@/hooks/useViewer";
import { APIError } from "@/lib/api";
import {
  createPage,
  getBrain,
  getPage,
  listPages,
  listPageTypes,
  updatePage,
} from "@/lib/brainApi";

type EditorError = "forbidden" | "not-found" | "load" | "";

export function PageEditorContainer() {
  const { sourceID = "", "*": editSlug = "" } = useParams<{
    sourceID: string;
    "*": string;
  }>();
  const editing = editSlug !== "";
  const navigate = useNavigate();
  const shell = useViewer();
  const [brain, setBrain] = useState<Brain | null>(null);
  const [pages, setPages] = useState<Page[]>([]);
  const [pageTypes, setPageTypes] = useState<PageType[]>([]);
  const [existingTimeline, setExistingTimeline] = useState("");
  const [slug, setSlug] = useState(editSlug);
  const [title, setTitle] = useState("");
  const [type, setType] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState("");
  const [supersededBy, setSupersededBy] = useState("");
  const [compiledTruth, setCompiledTruth] = useState("");
  const [timelineEntry, setTimelineEntry] = useState(
    editing ? "" : `${new Date().toISOString().slice(0, 10)} 初版`,
  );
  const [showTimelineEntry, setShowTimelineEntry] = useState(!editing);
  const [error, setError] = useState<EditorError>("");
  const [saveError, setSaveError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    document.title = `${editing ? editSlug : "ページを作成"} — ${sourceID} — brainhub`;
  }, [editSlug, editing, sourceID]);

  useEffect(() => {
    if (shell.viewer === undefined) return;
    if (shell.viewer === null) {
      setError("forbidden");
      return;
    }
    setError("");
    const requests: [Promise<Brain>, Promise<Page[]>, Promise<PageType[]>, Promise<PageDetail | null>] = [
      getBrain(sourceID),
      listPages(sourceID),
      listPageTypes(sourceID),
      editing ? getPage(sourceID, editSlug) : Promise.resolve(null),
    ];
    void Promise.all(requests)
      .then(([loadedBrain, loadedPages, loadedTypes, page]) => {
        setBrain(loadedBrain);
        setPages(loadedPages);
        setPageTypes(loadedTypes);
        setType(page?.type ?? loadedTypes[0]?.name ?? "");
        if (page) {
          setSlug(page.slug);
          setTitle(page.title);
          setTags(page.tags);
          setSupersededBy(page.superseded_by ?? "");
          setCompiledTruth(page.compiled_truth);
          setExistingTimeline(page.timeline);
        }
      })
      .catch((cause) => {
        if (cause instanceof APIError && (cause.status === 401 || cause.status === 403)) {
          setError("forbidden");
        } else if (cause instanceof APIError && cause.status === 404) {
          setError("not-found");
        } else {
          setError("load");
        }
      });
  }, [editSlug, editing, shell.viewer, sourceID]);

  const directories = useMemo(
    () =>
      [...new Set(pages.map((page) => page.slug.split("/").slice(0, -1).join("/")).filter(Boolean))]
        .sort()
        .map((directory) => `${directory}/`),
    [pages],
  );

  function addTag() {
    const tag = tagInput.trim();
    if (tag && !tags.includes(tag)) setTags((current) => [...current, tag]);
    setTagInput("");
  }

  async function save() {
    setSaving(true);
    setSaveError("");
    try {
      const input = {
        title,
        type,
        tags,
        superseded_by: supersededBy || null,
        compiled_truth: compiledTruth,
        timeline_entry: showTimelineEntry ? timelineEntry : "",
      };
      const saved = editing
        ? await updatePage(sourceID, editSlug, input)
        : await createPage(sourceID, { slug, ...input });
      navigate(pageUrl(sourceID, saved.slug));
    } catch (cause) {
      if (cause instanceof APIError && cause.status === 409) {
        setSaveError("同じ slug のページがすでに存在します。");
      } else if (cause instanceof APIError && cause.status === 400) {
        setSaveError("入力内容を確認してください。slug、type、参照先、Timeline はサーバーでも検証されます。");
      } else if (cause instanceof APIError && cause.status === 403) {
        setSaveError("この脳へ書き込む権限がありません。");
      } else {
        setSaveError("保存できませんでした。GBrain と writer client の状態を確認してください。");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <PageEditorPresenter
      shell={shell}
      sourceID={sourceID}
      brain={brain}
      editing={editing}
      error={error}
      pages={pages}
      pageTypes={pageTypes}
      directories={directories}
      slug={slug}
      title={title}
      type={type}
      tags={tags}
      tagInput={tagInput}
      supersededBy={supersededBy}
      compiledTruth={compiledTruth}
      existingTimeline={existingTimeline}
      timelineEntry={timelineEntry}
      showTimelineEntry={showTimelineEntry}
      saving={saving}
      saveError={saveError}
      onSlug={setSlug}
      onTitle={setTitle}
      onType={setType}
      onTagInput={setTagInput}
      onAddTag={addTag}
      onRemoveTag={(tag) => setTags((current) => current.filter((value) => value !== tag))}
      onSupersededBy={setSupersededBy}
      onCompiledTruth={setCompiledTruth}
      onTimelineEntry={setTimelineEntry}
      onShowTimelineEntry={() => setShowTimelineEntry(true)}
      onCancel={() => navigate(editing ? pageUrl(sourceID, editSlug) : brainUrl(sourceID))}
      onSave={save}
    />
  );
}
