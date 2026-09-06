export type BrainTab = "pages" | "connect" | "settings" | "invites";

export const appUrl = {
  brainList: "/",
  connections: "/settings/connections",
  login: "/login",
  register: "/register",
  createBrain: "/brains/new",
  brainDetail: "/brains/:sourceID",
  brainPage: "/brains/:sourceID/pages/*",
  invitation: "/invite/:token",
  oauthAuthorize: "/oauth/authorize",
} as const;

export function brainUrl(sourceID: string, tab: BrainTab = "pages"): string {
  if (tab === "connect") return appUrl.connections;
  const path = `/brains/${encodeURIComponent(sourceID)}`;
  return tab === "pages" ? path : `${path}?tab=${tab}`;
}

export function brainAddressPrefix(publicWebURL: string): string {
  const url = new URL(publicWebURL);
  const basePath = url.pathname.replace(/\/+$/, "");
  return `${url.host}${basePath}/brains/`;
}

export function invitationUrl(token: string): string {
  return `/invite/${encodeURIComponent(token)}`;
}

export function pageUrl(sourceID: string, slug: string): string {
  const encodedSlug = slug.split("/").map(encodeURIComponent).join("/");
  return `/brains/${encodeURIComponent(sourceID)}/pages/${encodedSlug}`;
}

export function markdownPageHref(sourceID: string, href?: string): string | undefined {
  if (!href || /^[a-z][a-z\d+.-]*:/i.test(href) || /^[/?#]/.test(href)) return href;
  const suffixAt = href.search(/[?#]/);
  const slug = (suffixAt < 0 ? href : href.slice(0, suffixAt)).replace(/^\.\//, "");
  return pageUrl(sourceID, slug) + (suffixAt < 0 ? "" : href.slice(suffixAt));
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
  if (tab === "settings") return "settings";
  return tab === "invites" ? "invites" : "pages";
}
