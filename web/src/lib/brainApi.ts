import type {
  Brain,
  CreateBrainInput,
  Page,
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

export function getPublicConfig(): Promise<PublicConfig> {
  return requestJSON("/config");
}

export function createBrain(input: CreateBrainInput): Promise<Brain> {
  return postJSON("/brains", input);
}
