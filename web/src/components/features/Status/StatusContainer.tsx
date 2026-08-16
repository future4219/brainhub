import { useEffect } from "react";

import { StatusPresenter } from "@/components/features/Status/StatusPresenter";
import { useViewer } from "@/hooks/useViewer";

export function StatusContainer({
  label,
  title,
}: {
  label: string;
  title: string;
}) {
  const shell = useViewer();

  useEffect(() => {
    document.title = `${label} — brainhub`;
  }, [label]);

  return <StatusPresenter {...shell} label={label} title={title} />;
}
