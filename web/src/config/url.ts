export type BrainTab = "pages" | "connect" | "invites";

export const appUrl = {
  brainList: "/",
  login: "/login",
  register: "/register",
  createBrain: "/brains/new",
  brainDetail: "/brains/:sourceID",
  brainPage: "/brains/:sourceID/pages/*",
  newPage: "/brains/:sourceID/page-editor/new",
  editPage: "/brains/:sourceID/page-editor/edit/*",
  invitation: "/invite/:token",
} as const;

export function brainUrl(sourceID: string, tab: BrainTab = "pages"): string {
  const path = `/brains/${encodeURIComponent(sourceID)}`;
  return tab === "pages" ? path : `${path}?tab=${tab}`;
}

export function invitationUrl(token: string): string {
  return `/invite/${encodeURIComponent(token)}`;
}

export function pageUrl(sourceID: string, slug: string): string {
  const encodedSlug = slug.split("/").map(encodeURIComponent).join("/");
  return `/brains/${encodeURIComponent(sourceID)}/pages/${encodedSlug}`;
}

export function newPageUrl(sourceID: string): string {
  return `/brains/${encodeURIComponent(sourceID)}/page-editor/new`;
}

export function editPageUrl(sourceID: string, slug: string): string {
  const encodedSlug = slug.split("/").map(encodeURIComponent).join("/");
  return `/brains/${encodeURIComponent(sourceID)}/page-editor/edit/${encodedSlug}`;
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
