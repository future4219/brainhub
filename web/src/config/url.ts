export type BrainTab = "pages" | "connect" | "invites";

export const appUrl = {
  brainList: "/",
  login: "/login",
  register: "/register",
  createBrain: "/brains/new",
  brainDetail: "/brains/:sourceID",
  invitation: "/invite/:token",
} as const;

export function brainUrl(sourceID: string, tab: BrainTab = "pages"): string {
  const path = `/brains/${encodeURIComponent(sourceID)}`;
  return tab === "pages" ? path : `${path}?tab=${tab}`;
}

export function invitationUrl(token: string): string {
  return `/invite/${encodeURIComponent(token)}`;
}

export function loginUrl(next?: string): string {
  return next
    ? `${appUrl.login}?next=${encodeURIComponent(next)}`
    : appUrl.login;
}

export function registerUrl(next?: string): string {
  return next
    ? `${appUrl.register}?next=${encodeURIComponent(next)}`
    : appUrl.register;
}

export function brainTab(search: string): BrainTab {
  const query = new URLSearchParams(search);
  const tab = query.get("tab");
  if (tab === "connect" || query.get("connect") === "1") return "connect";
  return tab === "invites" ? "invites" : "pages";
}
