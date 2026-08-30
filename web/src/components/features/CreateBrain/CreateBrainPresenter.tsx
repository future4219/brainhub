import type {
  ChangeEventHandler,
  FormEventHandler,
} from "react";
import { Link } from "react-router-dom";

import type { CreateBrainFailure } from "@/components/features/CreateBrain/createBrainForm";
import { AppShell } from "@/components/ui/AppShell";
import { Button, buttonVariants } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import { appUrl, loginUrl } from "@/config/url";
import type { ViewerState } from "@/entities/user/entity";

type CreateBrainPresenterProps = ViewerState & {
  name: string;
  sourceID: string;
  addressPrefix: string;
  configError: boolean;
  submitting: boolean;
  error: CreateBrainFailure | null;
  onNameChange: ChangeEventHandler<HTMLInputElement>;
  onSourceIDChange: ChangeEventHandler<HTMLInputElement>;
  onSubmit: FormEventHandler<HTMLFormElement>;
};

export function CreateBrainPresenter({
  name,
  sourceID,
  addressPrefix,
  configError,
  submitting,
  error,
  onNameChange,
  onSourceIDChange,
  onSubmit,
  ...shell
}: CreateBrainPresenterProps) {
  const sourceIDHint =
    name.trim() && !sourceID
      ? "半角英数字で入力してください"
      : "この脳のURLの末尾です。半角英数字とハイフンが使え、作成後は変更できません。";

  return (
    <AppShell {...shell} crumbs={["脳", "新しい脳"]}>
      <main className="mx-auto max-w-content px-4 pb-20 pt-8 sm:px-8">
        <div className="grid gap-8 md:grid-cols-2">
          <header>
            <Link
              className={buttonVariants({ variant: "ghost", size: "sm" })}
              to={appUrl.brainList}
            >
              ← 脳の一覧へ
            </Link>
            <h1 className="mt-4 text-title font-semibold tracking-tight">
              新しい脳を作る
            </h1>
            <p className="mt-4 max-w-credentials text-body text-text-secondary">
              名前、説明、公開範囲を設定します。
            </p>
          </header>

          <Panel>
            {(shell.viewer === undefined ||
              (shell.viewer && !addressPrefix && !configError)) && (
              <Feedback kind="loading">作成画面を準備中…</Feedback>
            )}
            {shell.viewer === null && (
              <div className="p-4">
                <Feedback kind="error" className="p-0 text-left">
                  脳を作るにはログインが必要です。ログイン後、この画面へ戻ってください。
                </Feedback>
                <Link
                  className={`${buttonVariants()} mt-6`}
                  to={loginUrl(appUrl.createBrain)}
                >
                  ログイン
                </Link>
              </div>
            )}
            {shell.viewer && configError && (
              <Feedback kind="error" className="text-left">
                公開URLを読み込めませんでした。接続を確認して、画面を再読み込みしてください。
              </Feedback>
            )}
            {shell.viewer && addressPrefix && (
              <form
                aria-busy={submitting}
                className="px-4 pb-6"
                onSubmit={onSubmit}
              >
                <FormField label="名前">
                  <Input
                    name="name"
                    maxLength={80}
                    placeholder="プロダクト調査"
                    value={name}
                    disabled={submitting}
                    onChange={onNameChange}
                    required
                  />
                </FormField>
                <label className="grid gap-2 border-b border-divider py-4">
                  <span className="font-mono text-xs uppercase tracking-label text-text-muted">
                    URL
                  </span>
                  <div className="flex h-control min-w-0 items-center overflow-hidden rounded-control border border-border-interactive bg-canvas focus-within:border-border-strong">
                    <span className="shrink-0 pl-3 font-mono text-ui text-text-secondary">
                      {addressPrefix}
                    </span>
                    <input
                      aria-describedby="source-id-hint"
                      autoCapitalize="none"
                      autoComplete="off"
                      className="min-w-0 flex-1 bg-transparent px-1 font-mono text-ui outline-none focus-visible:outline-none"
                      name="source_id"
                      pattern="[a-z0-9-]{1,32}"
                      maxLength={32}
                      placeholder="product-research"
                      spellCheck={false}
                      value={sourceID}
                      disabled={submitting}
                      onChange={onSourceIDChange}
                      required
                    />
                  </div>
                  <small
                    aria-live="polite"
                    className="text-xs text-text-muted"
                    id="source-id-hint"
                  >
                    {sourceIDHint}
                  </small>
                </label>
                <FormField label="説明（任意）">
                  <textarea
                    className="field-control h-auto min-h-0"
                    name="description"
                    maxLength={500}
                    rows={2}
                    disabled={submitting}
                  />
                </FormField>
                <FormField label="公開範囲">
                  <select
                    className="field-control"
                    name="visibility"
                    defaultValue="private"
                    disabled={submitting}
                  >
                    <option value="private">
                      private — 招待した人だけが見られる
                    </option>
                    <option value="public">public — 誰でも見られる</option>
                  </select>
                </FormField>
                {error && (
                  <Feedback kind="error" className="-mx-4 text-left">
                    {error.messages.map((message, index) => (
                      <p className={index ? "mt-2" : undefined} key={message}>
                        {message}
                      </p>
                    ))}
                    {error.showBrainList && (
                      <Link
                        className={`${buttonVariants({ variant: "outline", size: "sm" })} mt-4`}
                        to={appUrl.brainList}
                      >
                        脳の一覧を見る
                      </Link>
                    )}
                  </Feedback>
                )}
                <Button
                  className="mt-6 w-full text-[var(--bh-color-canvas)]"
                  type="submit"
                  disabled={submitting}
                >
                  <span>{submitting ? "作成中…" : "脳を作る"}</span>
                </Button>
              </form>
            )}
          </Panel>
        </div>
      </main>
    </AppShell>
  );
}
