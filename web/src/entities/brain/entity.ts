export type BrainState =
  | "provisioning"
  | "ready"
  | "degraded"
  | "failed"
  | "archived";

export type Brain = {
  id: string;
  source_id: string;
  name: string;
  description: string;
  visibility: "public" | "private";
  owner_id: string;
  state: BrainState;
  state_reason: string;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
};

export type Page = {
  slug: string;
  title: string;
  type: string;
  updated_at: string;
};

export type PageDetail = Page & {
  compiled_truth: string;
  timeline: string;
  tags: string[];
  superseded_by: string | null;
};

export type PageType = {
  name: string;
  primitive: string;
};

export type CreatePageInput = {
  slug: string;
  title: string;
  type: string;
  tags: string[];
  superseded_by: string | null;
  compiled_truth: string;
  timeline_entry: string;
};

export type UpdatePageInput = Omit<CreatePageInput, "slug">;

export type PublicConfig = { mcp_url: string };

export type CreateBrainInput = Pick<
  Brain,
  "source_id" | "name" | "description" | "visibility"
>;
