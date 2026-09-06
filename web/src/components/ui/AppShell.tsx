import type { ReactNode } from "react";
import { Link, useLocation, useParams } from "react-router-dom";

import { Button, buttonVariants } from "@/components/ui/Button";
import { appUrl, brainUrl } from "@/config/url";
import type { ViewerState } from "@/entities/user/entity";
import { cn } from "@/lib/utils";

type AppShellProps = ViewerState & {
  crumbs: string[];
  search?: { value: string; placeholder: string; onChange: (value: string) => void };
  onLogout: () => void;
  children: ReactNode;
};

export function AppShell({ viewer, brains, sessionUnavailable, crumbs, search, onLogout, children }: AppShellProps) {
  const { sourceID } = useParams<{ sourceID: string }>();
  const location = useLocation();
  const sidebarBrains = brains ?? [];

  return (
    <div className="flex min-h-screen bg-canvas text-text">
      <aside className="sticky top-0 hidden h-screen w-sidebar shrink-0 flex-col border-r border-border bg-sidebar md:flex">
        <Link className="flex items-center gap-2 px-4 pb-3 pt-4 font-mono text-body font-semibold tracking-tight" to={appUrl.brainList}>
          <span className="flex size-brand items-center justify-center rounded-brand bg-text text-xs text-canvas">bh</span>
          brainhub
        </Link>
        <nav className="px-2 py-1" aria-label="メイン">
          <Link className={cn("flex min-h-control items-center gap-2 rounded-control px-3 text-ui", location.pathname === appUrl.brainList && "bg-surface-selected")} to={appUrl.brainList}>
            <span className="font-mono text-xs text-text-muted">▤</span>
            <span className="flex-1">脳</span>
            <span className="font-mono text-xs text-text-muted">{sidebarBrains.length || ""}</span>
          </Link>
          <Link className={cn("flex min-h-control items-center rounded-control px-3 text-ui hover:bg-surface-selected", location.pathname === appUrl.connections && "bg-surface-selected")} to={appUrl.connections} aria-current={location.pathname === appUrl.connections ? "page" : undefined}>
            AIとの接続
          </Link>
        </nav>
        {sidebarBrains.length > 0 && (
          <div className="mt-4">
            <p className="px-4 pb-2 font-mono text-xs uppercase tracking-section text-text-muted">brains</p>
            <div className="px-2">
              {sidebarBrains.map((brain) => (
                <Link className={cn("flex min-h-control items-center gap-2 rounded-control px-3 font-mono text-sm text-text-secondary hover:bg-surface-selected hover:text-text", brain.source_id === sourceID && "bg-surface-selected text-text")} to={brainUrl(brain.source_id)} key={brain.id}>
                  <span className="sidebar-state-dot" data-state={brain.state} aria-hidden="true" />
                  <span className="min-w-0 flex-1 truncate">{brain.source_id}</span>
                  <span className="text-xs text-text-muted">[{brain.state}]</span>
                </Link>
              ))}
            </div>
          </div>
        )}
        <div className="flex-1" />
        <div className="p-2">
          {viewer === undefined ? (
            <div className="px-2 py-2 font-mono text-xs text-text-muted">sessionを確認中…</div>
          ) : viewer ? (
            <div className="rounded-control border border-border-control bg-surface p-2">
              <strong className="block truncate text-ui">{viewer.name}</strong>
              <span className="block truncate font-mono text-xs text-text-muted">{viewer.email}</span>
              <Button className="mt-2 w-full" variant="ghost" size="sm" type="button" onClick={onLogout}>ログアウト</Button>
            </div>
          ) : (
            <div className="flex gap-2 px-2 py-2 text-xs text-text-secondary">
              <Link className="hover:text-text-hover hover:underline" to={appUrl.login}>ログイン</Link>
              <Link className="hover:text-text-hover hover:underline" to={appUrl.register}>新規登録</Link>
            </div>
          )}
        </div>
      </aside>

      <div className="min-w-0 flex-1">
        <header className="topbar sticky top-0 z-20 flex h-topbar items-center gap-4 border-b border-border px-4 sm:px-8">
          <Link className="font-mono text-ui font-semibold md:hidden" to={appUrl.brainList}>brainhub</Link>
          <div className="hidden items-center gap-2 font-mono text-sm text-text-muted sm:flex">
            {[viewer?.name ?? "public", ...crumbs].map((crumb, index, values) => <span className={index === values.length - 1 ? "text-text" : undefined} key={`${crumb}-${index}`}>{index ? `/ ${crumb}` : crumb}</span>)}
          </div>
          <div className="flex-1" />
          <Link className="text-ui md:hidden" to={appUrl.connections}>AIとの接続</Link>
          {search && (
            <div className="shell-search hidden sm:flex">
              <span className="font-mono text-sm text-text-muted">/</span>
              <input aria-label={search.placeholder} value={search.value} placeholder={search.placeholder} onChange={(event) => search.onChange(event.target.value)} />
              <span className="shell-search-key">⌘K</span>
            </div>
          )}
          {viewer && <Link className={cn(buttonVariants({ size: "sm" }), "hidden sm:inline-flex")} to={appUrl.createBrain}>新しい脳</Link>}
          {sessionUnavailable && <span className="font-mono text-xs text-text-muted">session unavailable</span>}
          {!sessionUnavailable && viewer === null && (
            <nav className="flex gap-3 text-xs text-text-secondary md:hidden" aria-label="アカウント">
              <Link className="hover:text-text-hover hover:underline" to={appUrl.login}>ログイン</Link>
              <Link className="hover:text-text-hover hover:underline" to={appUrl.register}>新規登録</Link>
            </nav>
          )}
        </header>
        {children}
      </div>
    </div>
  );
}
