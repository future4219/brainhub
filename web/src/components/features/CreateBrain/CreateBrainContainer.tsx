import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { CreateBrainPresenter } from "@/components/features/CreateBrain/CreateBrainPresenter";
import { brainUrl } from "@/config/url";
import type { CreateBrainInput } from "@/entities/brain/entity";
import { useViewer } from "@/hooks/useViewer";
import { APIError } from "@/lib/api";
import { createBrain } from "@/lib/brainApi";

export function CreateBrainContainer() {
  const navigate = useNavigate();
  const shell = useViewer();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    document.title = "脳を作る — brainhub";
  }, []);

  async function submit(input: CreateBrainInput) {
    setSubmitting(true);
    setError("");
    try {
      const brain = await createBrain(input);
      navigate(brainUrl(brain.source_id), { replace: true });
    } catch (cause) {
      setError(createBrainErrorMessage(cause));
      setSubmitting(false);
    }
  }

  return (
    <CreateBrainPresenter
      {...shell}
      submitting={submitting}
      error={error}
      onSubmit={submit}
    />
  );
}

function createBrainErrorMessage(error: unknown) {
  if (!(error instanceof APIError)) {
    return "接続できません。時間を置いてもう一度お試しください。";
  }
  if (error.status === 401) {
    return "セッションが切れました。もう一度ログインしてください。";
  }
  if (error.status === 409) return "このsource IDはすでに使われています。";
  if (error.status === 502) {
    return "脳の登録に失敗しました。一覧で状態を確認してください。";
  }
  if (error.status === 400) {
    return "Source ID、名前、説明、公開範囲を確認してください。";
  }
  return "脳を作成できません。時間を置いてもう一度お試しください。";
}
