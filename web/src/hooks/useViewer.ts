import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { appUrl } from "@/config/url";
import type { User, ViewerState } from "@/entities/user/entity";
import { getCurrentUser, logout } from "@/lib/authApi";

export function useViewer(): ViewerState {
  const navigate = useNavigate();
  const [viewer, setViewer] = useState<User | null | undefined>(undefined);
  const [sessionUnavailable, setSessionUnavailable] = useState(false);

  useEffect(() => {
    void getCurrentUser().then(setViewer).catch(() => {
      setViewer(null);
      setSessionUnavailable(true);
    });
  }, []);

  async function handleLogout() {
    try {
      await logout();
      navigate(appUrl.brainList, { replace: true });
    } catch {
      window.alert("ログアウトできません。接続を確認してもう一度お試しください。");
    }
  }

  return { viewer, sessionUnavailable, onLogout: handleLogout };
}
