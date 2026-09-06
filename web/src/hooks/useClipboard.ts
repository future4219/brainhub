import { useState } from "react";

export type CopyState = Record<string, "copied" | "failed">;

export function useClipboard() {
  const [copyState, setCopyState] = useState<CopyState>({});

  async function copy(key: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopyState((current) => ({ ...current, [key]: "copied" }));
    } catch {
      setCopyState((current) => ({ ...current, [key]: "failed" }));
    }
  }

  function resetCopyState(key: string) {
    setCopyState((current) => {
      const next = { ...current };
      delete next[key];
      return next;
    });
  }

  return { copyState, copy, resetCopyState };
}
