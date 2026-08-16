import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import {
  AuthPresenter,
  type AuthMode,
} from "@/components/features/Auth/AuthPresenter";
import { appUrl, loginUrl, registerUrl } from "@/config/url";
import { useViewer } from "@/hooks/useViewer";
import { login, register } from "@/lib/authApi";
import { APIError } from "@/lib/api";
import { safeNextPath } from "@/lib/security";

export function AuthContainer({ mode }: { mode: AuthMode }) {
  const location = useLocation();
  const navigate = useNavigate();
  const shell = useViewer();
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const isRegister = mode === "register";
  const next = safeNextPath(
    location.search,
    isRegister ? appUrl.createBrain : appUrl.brainList,
  );

  useEffect(() => {
    document.title = `${isRegister ? "新規登録" : "ログイン"} — brainhub`;
  }, [isRegister]);

  async function submit(input: {
    email: string;
    password: string;
    name: string;
  }) {
    setSubmitting(true);
    setError("");
    try {
      if (isRegister) await register(input);
      else await login(input);
      navigate(next, { replace: true });
    } catch (cause) {
      setError(authErrorMessage(cause, mode));
      setSubmitting(false);
    }
  }

  return (
    <AuthPresenter
      {...shell}
      mode={mode}
      submitting={submitting}
      error={error}
      switchHref={isRegister ? loginUrl(next) : registerUrl(next)}
      onSubmit={submit}
    />
  );
}

function authErrorMessage(error: unknown, mode: AuthMode) {
  if (!(error instanceof APIError)) {
    return "接続できません。時間を置いてもう一度お試しください。";
  }
  if (error.status === 429) {
    return "ログイン試行が多すぎます。15分後にもう一度お試しください。";
  }
  if (error.status === 401) {
    return "メールアドレスまたはパスワードが一致しません。";
  }
  if (error.status === 409) {
    return "このメールアドレスはすでに登録されています。";
  }
  if (error.status === 400) {
    return mode === "register"
      ? "入力内容を確認してください。"
      : "メールアドレスとパスワードを入力してください。";
  }
  return "処理を完了できません。時間を置いてもう一度お試しください。";
}
