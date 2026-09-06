import { useEffect, useMemo, useState } from "react";
import { useLocation, useParams } from "react-router-dom";

import { BrainSettingsPresenter } from "@/components/features/BrainDetail/BrainSettingsPresenter";
import { BrainInvitationsPresenter } from "@/components/features/BrainDetail/BrainInvitationsPresenter";
import { BrainPagesPresenter } from "@/components/features/BrainDetail/BrainPagesPresenter";
import type {
  BrainDetailState,
  PageSort,
} from "@/components/features/BrainDetail/types";
import { brainTab, invitationUrl } from "@/config/url";
import type { Invitation, Role } from "@/entities/access/entity";
import type { Brain, Page } from "@/entities/brain/entity";
import { useClipboard } from "@/hooks/useClipboard";
import { useViewer } from "@/hooks/useViewer";
import {
  createInvitation,
  listInvitations,
  revokeInvitation,
} from "@/lib/accessApi";
import { APIError } from "@/lib/api";
import { getBrain, listPages, reissueWriter } from "@/lib/brainApi";

export function BrainDetailContainer() {
  const location = useLocation();
  const { sourceID = "" } = useParams<{ sourceID: string }>();
  const tab = brainTab(location.search);
  const shell = useViewer();
  const [brain, setBrain] = useState<Brain | null>(null);
  const [brainError, setBrainError] = useState<"not-found" | "load" | "">("");
  const [pages, setPages] = useState<Page[] | null>(null);
  const [pagesError, setPagesError] = useState("");
  const [invitations, setInvitations] = useState<Invitation[] | null>(null);
  const [inviteAuthorized, setInviteAuthorized] = useState<boolean | null>(
    null,
  );
  const [inviteError, setInviteError] = useState("");
  const [createdLink, setCreatedLink] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [writerReissuing, setWriterReissuing] = useState(false);
  const [writerStatus, setWriterStatus] = useState("");
  const { copyState, copy } = useClipboard();
  const [selectedType, setSelectedType] = useState("all");
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<PageSort>("updated");

  useEffect(() => {
    document.title = `${sourceID} — brainhub`;
    void getBrain(sourceID)
      .then(setBrain)
      .catch((cause) => {
        setBrainError(
          cause instanceof APIError && cause.status === 404
            ? "not-found"
            : "load",
        );
      });
  }, [sourceID]);

  useEffect(() => {
    if (brain?.state !== "ready") return;
    setPages(null);
    setPagesError("");
    void listPages(sourceID)
      .then(setPages)
      .catch(() =>
        setPagesError(
          "ページ一覧を取得できません。接続を確認して再読み込みしてください。",
        ),
      );
  }, [brain?.state, sourceID]);

  useEffect(() => {
    if (
      tab !== "invites" ||
      brain?.state !== "ready" ||
      shell.viewer === undefined
    ) {
      return;
    }
    if (shell.viewer === null) {
      setInvitations([]);
      setInviteAuthorized(false);
      return;
    }
    setInvitations(null);
    setInviteError("");
    void listInvitations(sourceID)
      .then((loaded) => {
        setInvitations(loaded);
        setInviteAuthorized(true);
      })
      .catch((cause) => {
        if (cause instanceof APIError && cause.status === 403) {
          setInvitations([]);
          setInviteAuthorized(false);
        } else {
          setInviteError(
            "招待一覧を取得できません。接続を確認して再読み込みしてください。",
          );
        }
      });
  }, [brain?.state, shell.viewer, sourceID, tab]);

  const typeCounts = useMemo(() => {
    const counts = new Map<string, number>();
    for (const page of pages ?? []) {
      counts.set(page.type, (counts.get(page.type) ?? 0) + 1);
    }
    return [...counts].sort(
      ([typeA, countA], [typeB, countB]) =>
        countB - countA || typeA.localeCompare(typeB),
    );
  }, [pages]);

  const visiblePages = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return (pages ?? [])
      .filter(
        (page) =>
          (selectedType === "all" || page.type === selectedType) &&
          (!normalized ||
            `${page.title} ${page.slug} ${page.type}`
              .toLowerCase()
              .includes(normalized)),
      )
      .sort((a, b) =>
        sort === "slug"
          ? a.slug.localeCompare(b.slug)
          : b.updated_at.localeCompare(a.updated_at),
      );
  }, [pages, query, selectedType, sort]);

  async function reissueBrainWriter() {
    setWriterReissuing(true);
    setWriterStatus("");
    try {
      await reissueWriter(sourceID);
      setWriterStatus("脳専用の接続を再発行しました。");
      setBrain(await getBrain(sourceID));
    } catch {
      setWriterStatus(
        "接続を再発行できませんでした。時間をおいて再試行してください。",
      );
    } finally {
      setWriterReissuing(false);
    }
  }

  async function refreshInvitations() {
    setInvitations(await listInvitations(sourceID));
    setInviteAuthorized(true);
  }

  async function createInvite(input: {
    email: string;
    role: Role;
    expires_at: string;
  }) {
    setSubmitting(true);
    setInviteError("");
    try {
      const created = await createInvitation(sourceID, {
        ...input,
        expires_at: new Date(input.expires_at).toISOString(),
      });
      setCreatedLink(
        new URL(invitationUrl(created.token), window.location.origin).href,
      );
      await refreshInvitations();
    } catch (cause) {
      setInviteError(
        cause instanceof APIError && cause.status === 400
          ? "メール、role、有効期限を確認してください。"
          : "招待を作成できません。状態を確認してもう一度お試しください。",
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function removeInvitation(id: string) {
    try {
      await revokeInvitation(id);
      await refreshInvitations();
    } catch {
      setInviteError(
        "招待を失効できません。状態を確認してもう一度お試しください。",
      );
    }
  }

  const detail: BrainDetailState = {
    sourceID,
    tab,
    brain,
    error: brainError,
  };

  if (tab === "pages") {
    return (
      <BrainPagesPresenter
        shell={shell}
        detail={detail}
        page={{
          sourceID,
          pages,
          error: pagesError,
          typeCounts,
          visiblePages,
          selectedType,
          query,
          sort,
          onType: setSelectedType,
          onQuery: setQuery,
          onSort: setSort,
        }}
      />
    );
  }

  if (tab === "settings") {
    return (
      <BrainSettingsPresenter
        shell={shell}
        detail={detail}
        canReissueWriter={
          shell.viewer?.id === brain?.owner_id && !!shell.viewer
        }
        writerReissuing={writerReissuing}
        writerStatus={writerStatus}
        onReissueWriter={reissueBrainWriter}
      />
    );
  }

  return (
    <BrainInvitationsPresenter
      shell={shell}
      detail={detail}
      pageCount={pages?.length}
      invitations={{
        viewer: shell.viewer,
        invitations,
        authorized: inviteAuthorized,
        error: inviteError,
        createdLink,
        submitting,
        copyState,
        onCopy: copy,
        onCreate: createInvite,
        onRevoke: removeInvitation,
      }}
    />
  );
}
