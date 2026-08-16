import { useEffect, useMemo, useState } from "react";

import { BrainListPresenter } from "@/components/features/BrainList/BrainListPresenter";
import type {
  BrainMetrics,
  Visibility,
} from "@/components/features/BrainList/types";
import type { Brain } from "@/entities/brain/entity";
import { useViewer } from "@/hooks/useViewer";
import { listBrains, listPages } from "@/lib/brainApi";

export function BrainListContainer() {
  const shell = useViewer();
  const [brains, setBrains] = useState<Brain[] | null>(null);
  const [error, setError] = useState(false);
  const [visibility, setVisibility] = useState<Visibility>("all");
  const [metrics, setMetrics] = useState<Record<string, BrainMetrics>>({});
  const [query, setQuery] = useState("");

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

  return (
    <BrainListPresenter
      {...shell}
      brains={brains}
      visibleBrains={visibleBrains}
      visibility={visibility}
      metrics={metrics}
      query={query}
      error={error}
      onVisibility={setVisibility}
      onQuery={setQuery}
    />
  );
}
