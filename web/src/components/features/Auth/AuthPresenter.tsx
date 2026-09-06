import { Link } from "react-router-dom";

import { AppShell } from "@/components/ui/AppShell";
import { Button } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import type { ViewerState } from "@/entities/user/entity";

export type AuthMode = "login" | "register";

type AuthPresenterProps = ViewerState & {
  mode: AuthMode;
  submitting: boolean;
  error: string;
  switchHref: string;
  onSubmit: (input: {
    email: string;
    password: string;
    name: string;
  }) => void;
};

export function AuthPresenter({
  mode,
  submitting,
  error,
  switchHref,
  onSubmit,
  ...shell
}: AuthPresenterProps) {
  const isRegister = mode === "register";
  return (
    <AppShell
      {...shell}
      crumbs={[isRegister ? "新規登録" : "ログイン"]}
    >
      <main className="mx-auto max-w-content px-4 pb-20 pt-8 sm:px-8">
        <div className="grid gap-8 md:grid-cols-2">
          <header>
            <p className="font-mono text-xs uppercase tracking-section text-text-muted">
              {isRegister ? "create account" : "welcome back"}
            </p>
            <h1 className="mt-2 text-title font-semibold tracking-tight">
              {isRegister ? "新規登録" : "ログイン"}
            </h1>
            <p className="mt-4 max-w-credentials text-body text-text-secondary">
              {isRegister
                ? "領域ごとの脳を作り、AIクライアントへ接続する。"
                : "所有している脳と接続状態を確認する。"}
            </p>
          </header>

          <Panel>
            {shell.viewer === undefined && (
              <Feedback kind="loading">sessionを確認中…</Feedback>
            )}
            {shell.viewer && (
              <Feedback kind="info">
                {shell.viewer.name}
                としてログイン中です。元の画面に戻ります。
              </Feedback>
            )}
            {shell.viewer === null && (
              <form
                className="px-4 pb-6"
                onSubmit={(event) => {
                  event.preventDefault();
                  const data = new FormData(event.currentTarget);
                  onSubmit({
                    email: String(data.get("email")),
                    password: String(data.get("password")),
                    name: String(data.get("name")),
                  });
                }}
              >
                {shell.sessionUnavailable && (
                  <Feedback kind="error" className="-mx-4">
                    セッション状態を取得できません。入力後にもう一度接続を試します。
                  </Feedback>
                )}
                {isRegister && (
                  <FormField label="名前">
                    <Input
                      name="name"
                      autoComplete="name"
                      maxLength={80}
                      required
                    />
                  </FormField>
                )}
                <FormField label="メールアドレス">
                  <Input
                    name="email"
                    type="email"
                    autoComplete="email"
                    required
                  />
                </FormField>
                <FormField
                  label="パスワード"
                  hint={isRegister ? "12〜72バイト" : undefined}
                >
                  <Input
                    name="password"
                    type="password"
                    autoComplete={
                      isRegister ? "new-password" : "current-password"
                    }
                    minLength={12}
                    maxLength={72}
                    required
                  />
                </FormField>
                {error && (
                  <Feedback kind="error" className="-mx-4">
                    {error}
                  </Feedback>
                )}
                <Button
                  className="mt-6"
                  type="submit"
                  disabled={submitting}
                >
                  {submitting
                    ? "処理中…"
                    : isRegister
                      ? "アカウントを作る"
                      : "ログイン"}
                </Button>
                <p className="mt-6 text-ui text-text-secondary">
                  {isRegister
                    ? "すでにアカウントがある場合は"
                    : "初めて使う場合は"}{" "}
                  <Link
                    className="text-text-code underline underline-offset-4 hover:text-text-hover"
                    to={switchHref}
                  >
                    {isRegister ? "ログイン" : "新規登録"}
                  </Link>
                </p>
              </form>
            )}
          </Panel>
        </div>
      </main>
    </AppShell>
  );
}
