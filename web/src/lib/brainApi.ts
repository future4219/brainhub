import type {
  Brain,
  CreateBrainInput,
  CreatePageInput,
  Page,
  PageDetail,
  PageType,
  PublicConfig,
  UpdatePageInput,
} from "@/entities/brain/entity";
import { postJSON, putJSON, requestJSON } from "@/lib/api";

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

export function listPageTypes(sourceID: string): Promise<PageType[]> {
  return requestJSON(`/brains/${encodeURIComponent(sourceID)}/page-types`);
}

export function createPage(
  sourceID: string,
  input: CreatePageInput,
): Promise<PageDetail> {
  return postJSON(`/brains/${encodeURIComponent(sourceID)}/pages`, input);
}

export function updatePage(
  sourceID: string,
  slug: string,
  input: UpdatePageInput,
): Promise<PageDetail> {
  const encodedSlug = slug.split("/").map(encodeURIComponent).join("/");
  return putJSON(
    `/brains/${encodeURIComponent(sourceID)}/pages/${encodedSlug}`,
    input,
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
