import { useEffect, useState } from "react";

import { ConnectionsPresenter } from "@/components/features/Connections/ConnectionsPresenter";
import type { PublicConfig } from "@/entities/brain/entity";
import type { CreatedMCPCLIToken, MCPConnection } from "@/entities/mcp/entity";
import { useViewer } from "@/hooks/useViewer";
import { getPublicConfig } from "@/lib/brainApi";
import {
  getMCPConnection,
  issueMCPClient,
  issueMCPCLIToken,
  reissueMCPReader,
  revokeMCPCLIToken,
} from "@/lib/mcpApi";

export function ConnectionsContainer() {
  const shell = useViewer();
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [connection, setConnection] = useState<MCPConnection | null>(null);
  const [connectError, setConnectError] = useState("");
  const [cliTokenLabel, setCLITokenLabel] = useState("");
  const [createdCLIToken, setCreatedCLIToken] =
    useState<CreatedMCPCLIToken | null>(null);
  const [cliTokenSubmitting, setCLITokenSubmitting] = useState(false);
  const [cliTokenRevoking, setCLITokenRevoking] = useState("");
  const [cliTokenError, setCLITokenError] = useState("");
  const [readerReissuing, setReaderReissuing] = useState(false);
  const [readerStatus, setReaderStatus] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [copyState, setCopyState] = useState<
    Record<string, "copied" | "failed">
  >({});

  useEffect(() => {
    document.title = "AIとの接続 — brainhub";
  }, []);
  useEffect(() => {
    let active = true;
    setConnectError("");
    setConnection(null);
    setCreatedCLIToken(null);
    void getPublicConfig()
      .then((value) => {
        if (active) setConfig(value);
      })
      .catch(() => {
        if (active)
          setConnectError("接続情報を読み込めません。再読み込みしてください。");
      });
    if (shell.viewer) {
      void getMCPConnection()
        .then((value) => {
          if (active) setConnection(value);
        })
        .catch(() => {
          if (active)
            setConnectError(
              "接続情報を読み込めません。再読み込みしてください。",
            );
        });
    }
    return () => {
      active = false;
    };
  }, [shell.viewer]);

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

  async function createClient(name: "claude-web" | "codex") {
    setSubmitting(true);
    setConnectError("");
    try {
      await issueMCPClient(name);
      await refreshConnection();
    } catch {
      setConnectError("接続を準備できません。もう一度お試しください。");
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
      setReaderStatus(
        "読み取り接続を再発行しました。AIからもう一度呼び出してください。",
      );
    } catch {
      setReaderStatus(
        "読み取り接続を再発行できません。表示された理由を確認して、もう一度実行してください。",
      );
    } finally {
      setReaderReissuing(false);
    }
  }

  return (
    <ConnectionsPresenter
      shell={shell}
      connectionProps={{
        viewer: shell.viewer,
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
        copyState,
        onCopy: copy,
        onIssue: createClient,
        onCLITokenLabel: setCLITokenLabel,
        onIssueCLIToken: createCLIToken,
        onRevokeCLIToken: revokeCLIToken,
        onReissueReader: reissueReader,
      }}
    />
  );
}
