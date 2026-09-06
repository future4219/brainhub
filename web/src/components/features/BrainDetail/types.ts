import type { BrainTab } from "@/config/url";
import type { Invitation, Role } from "@/entities/access/entity";
import type { Brain, Page } from "@/entities/brain/entity";
import type { ViewerState } from "@/entities/user/entity";

export type PageSort = "updated" | "slug";
export type CopyState = Record<string, "copied" | "failed">;

export type BrainDetailState = {
  sourceID: string;
  tab: BrainTab;
  brain: Brain | null;
  error: "not-found" | "load" | "";
};

export type PagesSectionProps = {
  sourceID: string;
  pages: Page[] | null;
  error: string;
  typeCounts: [string, number][];
  visiblePages: Page[];
  selectedType: string;
  query: string;
  sort: PageSort;
  onType: (type: string) => void;
  onQuery: (query: string) => void;
  onSort: (sort: PageSort) => void;
};

export type InvitationsSectionProps = {
  viewer: ViewerState["viewer"];
  invitations: Invitation[] | null;
  authorized: boolean | null;
  error: string;
  createdLink: string;
  submitting: boolean;
  copyState: CopyState;
  onCopy: (key: string, value: string) => void;
  onCreate: (input: { email: string; role: Role; expires_at: string }) => void;
  onRevoke: (id: string) => void;
};
