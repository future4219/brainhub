import { Link } from "react-router-dom";

import { AppShell } from "@/components/ui/AppShell";
import { Button, buttonVariants } from "@/components/ui/Button";
import { Feedback } from "@/components/ui/Feedback";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Panel } from "@/components/ui/Panel";
import { appUrl, loginUrl } from "@/config/url";
import type { CreateBrainInput } from "@/entities/brain/entity";
import type { ViewerState } from "@/entities/user/entity";

type CreateBrainPresenterProps = ViewerState & {
  submitting: boolean;
  error: string;
  onSubmit: (input: CreateBrainInput) => void;
};

export function CreateBrainPresenter({
  submitting,
  error,
  onSubmit,
  ...shell
}: CreateBrainPresenterProps) {
  return (
    <AppShell {...shell} crumbs={["脳", "新しい脳"]}>
      <main className="mx-auto max-w-content px-4 pb-20 pt-8 sm:px-8">
        <div className="grid gap-8 md:grid-cols-2">
          <header>
            <p className="font-mono text-xs uppercase tracking-section text-text-muted">
              new repository
            </p>
            <h1 className="mt-2 text-title font-semibold tracking-tight">
              脳を作る
            </h1>
            <p className="mt-4 max-w-credentials text-body text-text-secondary">
              ひとつの領域に、ひとつの脳。source
              IDは引用とURLに使われ、作成後も変わらない。
            </p>
          </header>

          <Panel>
            {shell.viewer === undefined && (
              <Feedback kind="loading">sessionを確認中…</Feedback>
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
            {shell.viewer && (
              <form
                className="px-4 pb-6"
                onSubmit={(event) => {
                  event.preventDefault();
                  const data = new FormData(event.currentTarget);
                  onSubmit({
                    source_id: String(data.get("source_id")),
                    name: String(data.get("name")),
                    description: String(data.get("description")),
                    visibility: String(
                      data.get("visibility"),
                    ) as CreateBrainInput["visibility"],
                  });
                }}
              >
                <FormField
                  label="Source ID"
                  hint="小文字・数字・ハイフン、1〜32文字。defaultは使用不可。"
                >
                  <Input
                    className="font-mono"
                    name="source_id"
                    pattern="[a-z0-9-]{1,32}"
                    maxLength={32}
                    placeholder="product-research"
                    required
                  />
                </FormField>
                <FormField label="名前">
                  <Input
                    name="name"
                    maxLength={80}
                    placeholder="Product research"
                    required
                  />
                </FormField>
                <FormField label="説明">
                  <textarea
                    className="field-control"
                    name="description"
                    maxLength={500}
                    rows={5}
                  />
                </FormField>
                <FormField label="公開範囲">
                  <select
                    className="field-control"
                    name="visibility"
                    defaultValue="private"
                  >
                    <option value="private">private — 所有者だけ</option>
                    <option value="public">public — 誰でも閲覧可能</option>
                  </select>
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
                  {submitting ? "作成中…" : "脳を作る"}
                </Button>
              </form>
            )}
          </Panel>
        </div>
      </main>
    </AppShell>
  );
}
