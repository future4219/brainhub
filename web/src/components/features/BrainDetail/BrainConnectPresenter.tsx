import { BrainDetailLayout } from "@/components/features/BrainDetail/BrainDetailLayout";
import { ConnectSection } from "@/components/features/BrainDetail/sections/ConnectSection";
import type {
  BrainDetailState,
  ConnectSectionProps,
} from "@/components/features/BrainDetail/types";
import type { ViewerState } from "@/entities/user/entity";

type BrainConnectPresenterProps = {
  shell: ViewerState;
  detail: BrainDetailState;
  connection: ConnectSectionProps;
};

export function BrainConnectPresenter({
  shell,
  detail,
  connection,
}: BrainConnectPresenterProps) {
  return (
    <BrainDetailLayout
      shell={shell}
      detail={detail}
      pageCount={connection.pageCount}
    >
      <ConnectSection {...connection} />
    </BrainDetailLayout>
  );
}
