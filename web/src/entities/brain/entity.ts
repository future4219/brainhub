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
};

export type PublicConfig = { mcp_url: string };

export type CreateBrainInput = Pick<
  Brain,
  "source_id" | "name" | "description" | "visibility"
>;
