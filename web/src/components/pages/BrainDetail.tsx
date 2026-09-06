import { Navigate, useLocation } from "react-router-dom";

import { BrainDetailContainer } from "@/components/features/BrainDetail/BrainDetailContainer";
import { appUrl, brainTab } from "@/config/url";

export function BrainDetail() {
  const location = useLocation();
  if (brainTab(location.search) === "connect")
    return <Navigate to={appUrl.connections} replace />;
  return <BrainDetailContainer />;
}
