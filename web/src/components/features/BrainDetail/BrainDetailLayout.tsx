import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import type { BrainDetailState } from "@/components/features/BrainDetail/types";
import { AppShell } from "@/components/ui/AppShell";
import { buttonVariants } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { Panel } from "@/components/ui/Panel";
import { StateBadge } from "@/components/ui/StateBadge";
import { brainUrl, type BrainTab } from "@/config/url";
import type { Brain } from "@/entities/brain/entity";
import type { ViewerState } from "@/entities/user/entity";
import { cn } from "@/lib/utils";

type BrainDetailLayoutProps = {
  shell: ViewerState;
  detail: BrainDetailState;
  pageCount?: number;
  inviteCount?: number;
  search?: {
    value: string;
    placeholder: string;
    onChange: (value: string) => void;
  };
  children: ReactNode;
};

export function BrainDetailLayout({
  shell,
  detail,
  pageCount,
  inviteCount,
  search,
  children,
}: BrainDetailLayoutProps) {
  return (
    <AppShell
      {...shell}
      crumbs={[
        detail.sourceID,
        detail.tab === "pages"
          ? "ページ"
          : detail.tab === "connect"
            ? "接続"
            : "招待",
      ]}
      search={search}
    >
      <main className="w-full px-4 pb-20 pt-8 sm:px-8">
        {detail.error && (
          <Feedback kind={detail.error === "not-found" ? "empty" : "error"}>
            {detail.error === "not-found"
              ? "この脳は見つからないか、閲覧権限がありません。脳の一覧へ戻ってください。"
              : "脳を読み込めません。接続を確認して再読み込みしてください。"}
          </Feedback>
        )}
        {!detail.error && !detail.brain && (
          <Feedback kind="loading">loading brain…</Feedback>
        )}
        {detail.brain && (
          <>
            <RepositoryHeader
              brain={detail.brain}
              sourceID={detail.sourceID}
              ownerName={
                shell.viewer?.id === detail.brain.owner_id
                  ? shell.viewer.name
                  : detail.brain.name
              }
            />
            <BrainTabs
              sourceID={detail.sourceID}
              tab={detail.tab}
              pageCount={pageCount}
              inviteCount={inviteCount}
            />
            {detail.brain.state !== "ready" ? (
              <Panel className="mt-6 p-4">
                <StateBadge state={detail.brain.state} />
                <p className="mt-3 text-body text-text-secondary">
                  この脳はまだ接続できません。状態がreadyになってからページ一覧を開けます。
                </p>
                {detail.brain.state_reason && (
                  <p className="mt-2 font-mono text-xs text-text-muted">
                    {detail.brain.state_reason}
                  </p>
                )}
              </Panel>
            ) : (
              children
            )}
          </>
        )}
      </main>
    </AppShell>
  );
}

function RepositoryHeader({
  brain,
  sourceID,
  ownerName,
}: {
  brain: Brain;
  sourceID: string;
  ownerName: string;
}) {
  return (
    <header className="flex flex-wrap items-start justify-between gap-6">
      <div>
        <div className="flex flex-wrap items-center gap-2">
          <h1 className="font-mono text-title font-semibold tracking-tight">
            {ownerName} / {sourceID}
          </h1>
          <span className="rounded-badge border border-border-strong px-2 py-0.5 font-mono text-xs text-text-secondary">
            {brain.visibility}
          </span>
          <StateBadge state={brain.state} />
        </div>
        {brain.description && (
          <p className="mt-3 max-w-copy text-body leading-copy text-text-secondary">
            {brain.description}
          </p>
        )}
      </div>
      <div className="flex gap-2">
        <Link
          className={buttonVariants({ variant: "outline", size: "sm" })}
          to={brainUrl(sourceID, "invites")}
        >
          招待
        </Link>
        <Link
          className={buttonVariants({ size: "sm" })}
          to={brainUrl(sourceID, "connect")}
        >
          この脳に繋ぐ
        </Link>
      </div>
    </header>
  );
}

function BrainTabs({
  sourceID,
  tab,
  pageCount,
  inviteCount,
}: {
  sourceID: string;
  tab: BrainTab;
  pageCount?: number;
  inviteCount?: number;
}) {
  return (
    <nav className="mt-6 flex gap-1 border-b border-border" aria-label="脳の画面">
      {(["pages", "connect", "invites"] as const).map((value) => (
        <Link
          className={cn(
            "flex items-center gap-2 border-b-2 px-3 py-3 text-ui",
            tab === value
              ? "border-text text-text"
              : "border-transparent text-text-muted hover:text-text",
          )}
          to={brainUrl(sourceID, value)}
          aria-current={tab === value ? "page" : undefined}
          key={value}
        >
          {value === "pages" ? "ページ" : value === "connect" ? "接続" : "招待"}
          {value === "pages" && pageCount !== undefined && (
            <span className="rounded-pill bg-surface px-2 font-mono text-xs text-text-secondary">
              {pageCount}
            </span>
          )}
          {value === "invites" && inviteCount !== undefined && (
            <span className="rounded-pill bg-surface px-2 font-mono text-xs text-text-secondary">
              {inviteCount}
            </span>
          )}
        </Link>
      ))}
    </nav>
  );
}
