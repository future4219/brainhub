import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { InvitationPresenter } from "@/components/features/Invitation/InvitationPresenter";
import { brainUrl } from "@/config/url";
import type { InvitationPreview } from "@/entities/access/entity";
import { useViewer } from "@/hooks/useViewer";
import { acceptInvitation, getInvitation } from "@/lib/accessApi";
import { APIError } from "@/lib/api";

export function InvitationContainer() {
  const navigate = useNavigate();
  const { token = "" } = useParams<{ token: string }>();
  const shell = useViewer();
  const [preview, setPreview] = useState<InvitationPreview | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    document.title = "招待 — brainhub";
    void getInvitation(token)
      .then(setPreview)
      .catch((cause) => {
        if (cause instanceof APIError && cause.status === 404) {
          setNotFound(true);
        } else {
          setError(
            "招待を読み込めません。接続を確認して再読み込みしてください。",
          );
        }
      });
  }, [token]);

  async function accept() {
    setSubmitting(true);
    setError("");
    try {
      const accepted = await acceptInvitation(token);
      navigate(brainUrl(accepted.source_id, "connect"), { replace: true });
    } catch (cause) {
      if (cause instanceof APIError && cause.status === 409) {
        setError(
          "この招待はすでに受諾済みです。脳の一覧から接続してください。",
        );
      } else if (cause instanceof APIError && cause.status === 403) {
        setError(
          "この招待に指定されたメールアドレスと一致しません。別のアカウントでログインしてください。",
        );
      } else if (cause instanceof APIError && cause.status === 404) {
        setNotFound(true);
      } else {
        setError(
          "招待を受諾できません。接続を確認してもう一度お試しください。",
        );
      }
      setSubmitting(false);
    }
  }

  return (
    <InvitationPresenter
      {...shell}
      token={token}
      preview={preview}
      notFound={notFound}
      error={error}
      submitting={submitting}
      onAccept={accept}
    />
  );
}
