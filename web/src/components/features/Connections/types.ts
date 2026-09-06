import type { CopyState } from "@/hooks/useClipboard";
import type { PublicConfig } from "@/entities/brain/entity";
import type { CreatedMCPCLIToken, MCPConnection } from "@/entities/mcp/entity";
import type { ViewerState } from "@/entities/user/entity";

export type ConnectSectionProps = {
  viewer: ViewerState["viewer"];
  config: PublicConfig | null;
  connection: MCPConnection | null;
  error: string;
  cliTokenLabel: string;
  createdCLIToken: CreatedMCPCLIToken | null;
  cliTokenSubmitting: boolean;
  cliTokenRevoking: string;
  cliTokenError: string;
  submitting: boolean;
  readerReissuing: boolean;
  readerStatus: string;
  copyState: CopyState;
  onCopy: (key: string, value: string) => void;
  onIssue: (name: "claude-web" | "codex") => void;
  onCLITokenLabel: (label: string) => void;
  onIssueCLIToken: () => void;
  onRevokeCLIToken: (id: string) => void;
  onReissueReader: () => void;
};
