export type Brain = {
  id: string;
  name: string;
  local_path: string | null;
  remote_url: string | null;
  federated: boolean;
  page_count: number;
  last_sync_at: string | null;
};

export type Page = {
  slug: string;
  title: string;
  type: string;
  updated_at: string;
};

export type PublicConfig = {
  mcp_url: string;
};

export class APIError extends Error {
  constructor(public readonly status: number) {
    super(`API returned HTTP ${status}`);
  }
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, { headers: { Accept: "application/json" } });
  if (!response.ok) {
    throw new APIError(response.status);
  }
  return response.json() as Promise<T>;
}

export async function listBrains(): Promise<Brain[]> {
  const response = await getJSON<{ sources: Brain[] }>("/api/brains");
  return response.sources;
}

export function listPages(sourceID: string): Promise<Page[]> {
  return getJSON(`/api/brains/${encodeURIComponent(sourceID)}/pages`);
}

export function getPublicConfig(): Promise<PublicConfig> {
  return getJSON("/api/config");
}
