import { useEffect, useState } from "react";
import { useLocation } from "react-router-dom";

import { OAuthConsentPresenter } from "@/components/features/OAuthConsent/OAuthConsentPresenter";
import type { OAuthConsent } from "@/entities/mcp/entity";
import { useViewer } from "@/hooks/useViewer";
import { decideOAuth, getOAuthConsent } from "@/lib/mcpApi";

export function OAuthConsentContainer() {
  const location = useLocation();
  const shell = useViewer();
  const [consent, setConsent] = useState<OAuthConsent | null>(null);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    document.title = "接続を許可 — brainhub";
  }, []);

  useEffect(() => {
    setConsent(null);
    if (!shell.viewer) return;
    let active = true;
    setError("");
    void getOAuthConsent(location.search)
      .then((value) => {
        if (active) setConsent(value);
      })
      .catch(() => {
        if (active)
          setError(
            "接続の確認に失敗しました。AIとの接続画面からやり直してください。",
          );
      });
    return () => {
      active = false;
    };
  }, [location.search, shell.viewer]);

  async function decide(decision: "approve" | "deny") {
    setSubmitting(true);
    setError("");
    try {
      const result = await decideOAuth(location.search, decision);
      window.location.assign(result.redirect_uri);
    } catch {
      setError(
        "認可を完了できません。接続画面から新しいClient IDを確認し、もう一度実行してください。",
      );
      setSubmitting(false);
    }
  }

  return (
    <OAuthConsentPresenter
      {...shell}
      consent={consent}
      error={error}
      submitting={submitting}
      loginNext={`${location.pathname}${location.search}`}
      onApprove={() => void decide("approve")}
      onDeny={() => void decide("deny")}
    />
  );
}
