export class APIError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    public readonly data?: unknown,
  ) {
    super(message);
  }
}

export async function requestJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api${path}`, {
    ...init,
    credentials: "same-origin",
    headers: { Accept: "application/json", ...init?.headers },
  });
  if (!response.ok) {
    const text = (await response.text()).trim();
    let message = text || response.statusText;
    let data: unknown;
    try {
      data = JSON.parse(text) as unknown;
      const error = (data as { error?: unknown }).error;
      if (typeof error === "string") message = error;
    } catch {
      // Some API errors are intentionally plain text.
    }
    throw new APIError(response.status, message, data);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}

export function postJSON<T>(path: string, body?: unknown): Promise<T> {
  return requestJSON(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}

export function putJSON<T>(path: string, body: unknown): Promise<T> {
  return requestJSON(path, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}
