import { useEffect, useMemo, useRef, useState } from "react";

import { BrainListPresenter } from "@/components/features/BrainList/BrainListPresenter";
import type {
  BrainMetrics,
  Visibility,
} from "@/components/features/BrainList/types";
import type { Brain } from "@/entities/brain/entity";
import { useViewer } from "@/hooks/useViewer";
import { APIError } from "@/lib/api";
import { archiveBrain, listBrains, listPages } from "@/lib/brainApi";

export function BrainListContainer() {
  const shell = useViewer();
  const [brains, setBrains] = useState<Brain[] | null>(null);
  const [error, setError] = useState(false);
  const [visibility, setVisibility] = useState<Visibility>("all");
  const [metrics, setMetrics] = useState<Record<string, BrainMetrics>>({});
  const [query, setQuery] = useState("");
  const deletingRef = useRef(false);
  const [deletingSourceID, setDeletingSourceID] = useState("");
  const [deleteError, setDeleteError] = useState("");

  useEffect(() => {
    document.title = "brainhub — brains";
    void listBrains()
      .then(async (loaded) => {
        setBrains(loaded);
        const entries = await Promise.all(
          loaded.map(async (brain) => {
            if (brain.state !== "ready") {
              return [brain.source_id, { pages: null }] as const;
            }
            try {
              return [
                brain.source_id,
                { pages: await listPages(brain.source_id) },
              ] as const;
            } catch {
              return [brain.source_id, { pages: null }] as const;
            }
          }),
        );
        setMetrics(Object.fromEntries(entries));
      })
      .catch(() => setError(true));
  }, []);

  const visibleBrains = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return (
      brains?.filter(
        (brain) =>
          (visibility === "all" || brain.visibility === visibility) &&
          (!normalized ||
            `${brain.source_id} ${brain.name} ${brain.description}`
              .toLowerCase()
              .includes(normalized)),
      ) ?? []
    );
  }, [brains, query, visibility]);

  async function removeBrain(brain: Brain) {
    if (
      deletingRef.current ||
      !window.confirm(
        `「${brain.source_id}」を削除します。\nページは一覧から見えなくなり、接続は失効します。このURLは再利用できません。`,
      )
    ) {
      return;
    }
    deletingRef.current = true;
    setDeletingSourceID(brain.source_id);
    setDeleteError("");
    try {
      await archiveBrain(brain.source_id);
      setBrains((current) =>
        current?.filter((item) => item.id !== brain.id) ?? null,
      );
    } catch (cause) {
      setDeleteError(deleteBrainError(cause));
    } finally {
      deletingRef.current = false;
      setDeletingSourceID("");
    }
  }

  return (
    <BrainListPresenter
      {...shell}
      brains={brains}
      visibleBrains={visibleBrains}
      visibility={visibility}
      metrics={metrics}
      query={query}
      error={error}
      deletingSourceID={deletingSourceID}
      deleteError={deleteError}
      onVisibility={setVisibility}
      onQuery={setQuery}
      onDelete={removeBrain}
    />
  );
}

function deleteBrainError(cause: unknown): string {
  if (cause instanceof APIError && cause.status === 502) {
    return "接続を失効できなかったため、脳は削除されていません。GBrainの接続を確認して、もう一度削除してください。";
  }
  if (cause instanceof APIError && cause.status === 403) {
    return "この脳を削除できるのは所有者だけです。";
  }
  return "脳を削除できませんでした。状態を確認して、もう一度削除してください。";
}
