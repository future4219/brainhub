import type { InvitationsSectionProps } from "@/components/features/BrainDetail/types";
import { copyLabel } from "@/lib/format";
import { Button } from "@/components/ui/Button";
import { CopyValue } from "@/components/ui/CopyValue";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import { StateBadge } from "@/components/ui/StateBadge";
import type { Role } from "@/entities/access/entity";
import { formatDate } from "@/lib/format";

export function InvitationsSection(props: InvitationsSectionProps) {
  if (props.viewer === undefined) {
    return (
      <Feedback kind="loading" className="mt-6">
        sessionを確認中…
      </Feedback>
    );
  }
  if (props.viewer === null) {
    return (
      <Feedback kind="empty" className="mt-6">
        招待を管理するにはログインしてください。ログイン後、この脳の招待タブへ戻ってください。
      </Feedback>
    );
  }
  if (props.authorized === false) {
    return (
      <Feedback kind="empty" className="mt-6">
        招待を管理できるのは脳の所有者だけです。所有者へ依頼してください。
      </Feedback>
    );
  }

  return (
    <div className="mt-6 space-y-6">
      <header>
        <h2 className="text-section font-semibold">招待</h2>
        <p className="mt-2 max-w-copy text-body leading-copy text-text-secondary">
          public な脳では書き込み権限を、private
          な脳では閲覧または書き込み権限を渡す。
        </p>
      </header>
      <Panel className="p-4">
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(event) => {
            event.preventDefault();
            const data = new FormData(event.currentTarget);
            props.onCreate({
              email: String(data.get("email")),
              role: String(data.get("role")) as Role,
              expires_at: String(data.get("expires_at")),
            });
            event.currentTarget.reset();
          }}
        >
          <FormField
            className="min-w-0 flex-1 basis-64 border-0 py-0"
            label="宛先"
          >
            <Input
              name="email"
              type="email"
              placeholder="メールアドレス、または空欄でリンクを発行"
            />
          </FormField>
          <FormField className="basis-32 border-0 py-0" label="権限">
            <select className="field-control" name="role" defaultValue="reader">
              <option value="reader">reader</option>
              <option value="editor">editor</option>
              <option value="owner">owner</option>
            </select>
          </FormField>
          <FormField className="basis-48 border-0 py-0" label="有効期限">
            <Input name="expires_at" type="datetime-local" required />
          </FormField>
          <Button type="submit" disabled={props.submitting}>
            {props.submitting ? "作成中…" : "招待を作る"}
          </Button>
        </form>
        <p className="mt-3 font-mono text-xs text-text-muted">
          発行後に出る /invite/{"{token}"}
          を相手に渡す。受諾した時点で接続できるようになる。
        </p>
        {props.createdLink && (
          <div className="mt-4 border-t border-divider">
            <CopyValue
              label="招待リンク"
              value={props.createdLink}
              note="token は作成時だけ表示される。"
              copyLabel={copyLabel(props.copyState.invite)}
              onCopy={() => props.onCopy("invite", props.createdLink)}
            />
          </div>
        )}
        {props.copyState.invite === "failed" && (
          <Feedback kind="error">
            コピーできません。リンクを選択してコピーしてください。
          </Feedback>
        )}
      </Panel>
      <Panel>
        <div className="border-b border-border px-4 py-3 font-mono text-xs text-text-secondary">
          {props.invitations?.length ?? 0} invites · pending{" "}
          {props.invitations?.filter(
            (invitation) => invitation.state === "pending",
          ).length ?? 0}
        </div>
        {props.error && (
          <Feedback kind="error">{props.error}</Feedback>
        )}
        {!props.error && props.invitations === null && (
          <Feedback kind="loading">loading invitations…</Feedback>
        )}
        {!props.error && props.invitations?.length === 0 && (
          <Feedback kind="empty">
            招待はまだありません。上のフォームから最初の招待を作成してください。
          </Feedback>
        )}
        {props.invitations && props.invitations.length > 0 && (
          <div className="overflow-x-auto">
            <div
              className="invite-table-grid grid min-w-credentials gap-4 border-b border-border bg-table-head px-4 py-2 font-mono text-xs uppercase tracking-label text-text-muted"
              aria-hidden="true"
            >
              <span>invitee</span>
              <span>role</span>
              <span>state</span>
              <span>expires</span>
              <span />
            </div>
            {props.invitations.map((invitation) => (
              <div
                className="invite-table-grid grid min-w-credentials items-center gap-4 border-b border-divider px-4 py-3 last:border-b-0 hover:bg-surface"
                key={invitation.id}
              >
                <span className="truncate font-mono text-sm">
                  {invitation.email || "リンクを知る人"}
                </span>
                <span className="font-mono text-xs text-text-secondary">
                  {invitation.role}
                </span>
                <StateBadge state={invitation.state} />
                <time
                  className="font-mono text-xs text-text-muted"
                  dateTime={invitation.expires_at}
                >
                  {formatDate(invitation.expires_at)}
                </time>
                <Button
                  variant="outline"
                  size="sm"
                  type="button"
                  disabled={invitation.state !== "pending"}
                  onClick={() => props.onRevoke(invitation.id)}
                >
                  失効
                </Button>
              </div>
            ))}
          </div>
        )}
      </Panel>
    </div>
  );
}
