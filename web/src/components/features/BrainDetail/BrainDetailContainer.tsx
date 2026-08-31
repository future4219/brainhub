import { useEffect, useMemo, useState } from "react";
import { useLocation, useParams } from "react-router-dom";

import { BrainConnectPresenter } from "@/components/features/BrainDetail/BrainConnectPresenter";
import { BrainInvitationsPresenter } from "@/components/features/BrainDetail/BrainInvitationsPresenter";
import { BrainPagesPresenter } from "@/components/features/BrainDetail/BrainPagesPresenter";
import type { BrainDetailState, PageSort } from "@/components/features/BrainDetail/types";
import { brainTab, invitationUrl } from "@/config/url";
import type {
  Invitation,
  Role,
} from "@/entities/access/entity";
import type { Brain, Page, PublicConfig } from "@/entities/brain/entity";
import type {
  CreatedMCPCLIToken,
  MCPConnection,
} from "@/entities/mcp/entity";
import { useViewer } from "@/hooks/useViewer";
import {
  createInvitation,
  listInvitations,
  revokeInvitation,
} from "@/lib/accessApi";
import { APIError } from "@/lib/api";
import {
  getBrain,
  getPublicConfig,
  listPages,
  listPageTypes,
  reissueWriter,
} from "@/lib/brainApi";
import {
  getMCPConnection,
  issueMCPClient,
  issueMCPCLIToken,
  reissueMCPReader,
  revokeMCPCLIToken,
} from "@/lib/mcpApi";

export function BrainDetailContainer() {
  const location = useLocation();
  const { sourceID = "" } = useParams<{ sourceID: string }>();
  const tab = brainTab(location.search);
  const shell = useViewer();
  const [brain, setBrain] = useState<Brain | null>(null);
  const [brainError, setBrainError] = useState<"not-found" | "load" | "">(
    "",
  );
  const [pages, setPages] = useState<Page[] | null>(null);
  const [pagesError, setPagesError] = useState("");
  const [canWrite, setCanWrite] = useState(false);
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [connection, setConnection] = useState<MCPConnection | null>(null);
  const [connectError, setConnectError] = useState("");
  const [cliTokenLabel, setCLITokenLabel] = useState("");
  const [createdCLIToken, setCreatedCLIToken] =
    useState<CreatedMCPCLIToken | null>(null);
  const [cliTokenSubmitting, setCLITokenSubmitting] = useState(false);
  const [cliTokenRevoking, setCLITokenRevoking] = useState("");
  const [cliTokenError, setCLITokenError] = useState("");
  const [invitations, setInvitations] = useState<Invitation[] | null>(null);
  const [inviteAuthorized, setInviteAuthorized] = useState<boolean | null>(null);
  const [inviteError, setInviteError] = useState("");
  const [createdLink, setCreatedLink] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [writerReissuing, setWriterReissuing] = useState(false);
  const [writerStatus, setWriterStatus] = useState("");
  const [readerReissuing, setReaderReissuing] = useState(false);
  const [readerStatus, setReaderStatus] = useState("");
  const [copyState, setCopyState] = useState<
    Record<string, "copied" | "failed">
  >({});
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
    if (brain?.state !== "ready" || shell.viewer === undefined) return;
    if (shell.viewer === null) {
      setCanWrite(false);
      return;
    }
    void listPageTypes(sourceID)
      .then(() => setCanWrite(true))
      .catch(() => setCanWrite(false));
  }, [brain?.state, shell.viewer, sourceID]);

  useEffect(() => {
    if (tab !== "connect" || brain?.state !== "ready") return;
    setConnectError("");
    void getPublicConfig()
      .then(setConfig)
      .catch(() =>
        setConnectError(
          "接続情報を取得できません。接続を確認して再読み込みしてください。",
        ),
      );
    if (shell.viewer === undefined) return;
    if (shell.viewer === null) {
      setConnection(null);
      return;
    }
    setConnection(null);
    void getMCPConnection()
      .then(setConnection)
      .catch(() =>
        setConnectError(
          "接続情報を取得できません。接続を確認して再読み込みしてください。",
        ),
      );
  }, [brain?.state, shell.viewer, sourceID, tab]);

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

  async function copy(key: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopyState((current) => ({ ...current, [key]: "copied" }));
    } catch {
      setCopyState((current) => ({ ...current, [key]: "failed" }));
    }
  }

  async function refreshConnection() {
    setConnection(await getMCPConnection());
  }

  async function createClient() {
    setSubmitting(true);
    setConnectError("");
    try {
      await issueMCPClient();
      await refreshConnection();
    } catch {
      setConnectError(
        "Claude用Client IDを発行できません。状態を確認してもう一度お試しください。",
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function createCLIToken() {
    setCLITokenSubmitting(true);
    setCLITokenError("");
    try {
      const issued = await issueMCPCLIToken(cliTokenLabel);
      setCreatedCLIToken(issued);
      setCLITokenLabel("");
      setCopyState((current) => {
        const next = { ...current };
        delete next["cli-token"];
        return next;
      });
      await refreshConnection();
    } catch {
      setCLITokenError(
        "CLI用トークンを発行できません。ラベルを確認して、もう一度発行してください。",
      );
    } finally {
      setCLITokenSubmitting(false);
    }
  }

  async function revokeCLIToken(id: string) {
    setCLITokenRevoking(id);
    setCLITokenError("");
    try {
      await revokeMCPCLIToken(id);
      if (createdCLIToken?.id === id) setCreatedCLIToken(null);
      await refreshConnection();
    } catch {
      setCLITokenError(
        "CLI用トークンを失効できません。再読み込みして、もう一度実行してください。",
      );
    } finally {
      setCLITokenRevoking("");
    }
  }

  async function reissueReader() {
    setReaderReissuing(true);
    setReaderStatus("");
    try {
      await reissueMCPReader();
      await refreshConnection();
      setReaderStatus("読み取り接続を再発行しました。Claudeからもう一度呼び出してください。");
    } catch {
      setReaderStatus(
        "読み取り接続を再発行できません。表示された理由を確認して、もう一度実行してください。",
      );
    } finally {
      setReaderReissuing(false);
    }
  }

  async function reissueBrainWriter() {
    setWriterReissuing(true);
    setWriterStatus("");
    try {
      await reissueWriter(sourceID);
      setWriterStatus("Writer client を再発行しました。ページを書き込めます。");
      setBrain(await getBrain(sourceID));
      setCanWrite(true);
    } catch {
      setWriterStatus(
        "Writer client を再発行できません。GBrain と暗号鍵の設定を確認してください。",
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
          canWrite,
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

  if (tab === "connect") {
    return (
      <BrainConnectPresenter
        shell={shell}
        detail={detail}
        connection={{
          sourceID,
          viewer: shell.viewer,
          brain,
          pageCount: pages?.length,
          config,
          connection,
          error: connectError,
          cliTokenLabel,
          createdCLIToken,
          cliTokenSubmitting,
          cliTokenRevoking,
          cliTokenError,
          submitting,
          readerReissuing,
          readerStatus,
          writerReissuing,
          writerStatus,
          canReissueWriter:
            shell.viewer !== null &&
            shell.viewer !== undefined &&
            shell.viewer.id === brain?.owner_id,
          copyState,
          onCopy: copy,
          onIssue: createClient,
          onCLITokenLabel: setCLITokenLabel,
          onIssueCLIToken: createCLIToken,
          onRevokeCLIToken: revokeCLIToken,
          onReissueReader: reissueReader,
          onReissueWriter: reissueBrainWriter,
        }}
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
