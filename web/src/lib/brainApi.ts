import type {
  Brain,
  CreateBrainInput,
  Page,
  PageDetail,
  PublicConfig,
} from "@/entities/brain/entity";
import { postJSON, requestJSON } from "@/lib/api";

export function listBrains(): Promise<Brain[]> {
  return requestJSON("/brains");
}

export function getBrain(sourceID: string): Promise<Brain> {
  return requestJSON(`/brains/${encodeURIComponent(sourceID)}`);
}

export function listPages(sourceID: string): Promise<Page[]> {
  return requestJSON(`/brains/${encodeURIComponent(sourceID)}/pages`);
}

export function getPage(sourceID: string, slug: string): Promise<PageDetail> {
  const encodedSlug = slug.split("/").map(encodeURIComponent).join("/");
  return requestJSON(
    `/brains/${encodeURIComponent(sourceID)}/pages/${encodedSlug}`,
  );
}

export function reissueWriter(sourceID: string): Promise<void> {
  return postJSON(`/brains/${encodeURIComponent(sourceID)}/writer/reissue`);
}

export function getPublicConfig(): Promise<PublicConfig> {
  return requestJSON("/config");
}

export function createBrain(input: CreateBrainInput): Promise<Brain> {
  return postJSON("/brains", input);
}

export function archiveBrain(sourceID: string): Promise<void> {
  return requestJSON(`/brains/${encodeURIComponent(sourceID)}`, {
    method: "DELETE",
  });
}
