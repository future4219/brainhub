import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import { BrainPagePresenter } from "@/components/features/BrainDetail/BrainPagePresenter";
import type { Brain, PageDetail } from "@/entities/brain/entity";
import { useViewer } from "@/hooks/useViewer";
import { APIError } from "@/lib/api";
import { getBrain, getPage, listPageTypes } from "@/lib/brainApi";

export function BrainPageContainer() {
  const { sourceID = "", "*": slug = "" } = useParams<{
    sourceID: string;
    "*": string;
  }>();
  const shell = useViewer();
  const [brain, setBrain] = useState<Brain | null>(null);
  const [brainError, setBrainError] = useState<"not-found" | "load" | "">("");
  const [page, setPage] = useState<PageDetail | null>(null);
  const [pageError, setPageError] = useState<"not-found" | "load" | "">("");
  const [canWrite, setCanWrite] = useState(false);

  useEffect(() => {
    document.title = `${page?.title ?? slug} — ${sourceID} — brainhub`;
  }, [page?.title, slug, sourceID]);

  useEffect(() => {
    setBrain(null);
    setBrainError("");
    void getBrain(sourceID)
      .then(setBrain)
      .catch((cause) =>
        setBrainError(
          cause instanceof APIError && cause.status === 404
            ? "not-found"
            : "load",
        ),
      );
  }, [sourceID]);

  useEffect(() => {
    setPage(null);
    setPageError("");
    void getPage(sourceID, slug)
      .then(setPage)
      .catch((cause) =>
        setPageError(
          cause instanceof APIError && cause.status === 404
            ? "not-found"
            : "load",
        ),
      );
  }, [slug, sourceID]);

  useEffect(() => {
    if (shell.viewer === undefined) return;
    if (shell.viewer === null) {
      setCanWrite(false);
      return;
    }
    void listPageTypes(sourceID)
      .then(() => setCanWrite(true))
      .catch(() => setCanWrite(false));
  }, [shell.viewer, sourceID]);

  return (
    <BrainPagePresenter
      shell={shell}
      sourceID={sourceID}
      brain={brain}
      brainError={brainError}
      page={page}
      pageError={pageError}
      canWrite={canWrite}
    />
  );
}
