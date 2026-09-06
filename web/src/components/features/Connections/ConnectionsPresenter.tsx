import { ConnectSection } from "@/components/features/Connections/ConnectSection";
import type { ConnectSectionProps } from "@/components/features/Connections/types";
import { AppShell } from "@/components/ui/AppShell";
import type { ViewerState } from "@/entities/user/entity";

export function ConnectionsPresenter({
  shell,
  connectionProps,
}: {
  shell: ViewerState;
  connectionProps: ConnectSectionProps;
}) {
  return (
    <AppShell {...shell} crumbs={["AIとの接続"]}>
      <main className="w-full px-4 pb-20 pt-8 sm:px-8">
        <ConnectSection {...connectionProps} />
      </main>
    </AppShell>
  );
}
