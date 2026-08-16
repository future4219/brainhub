import { BrainDetailLayout } from "@/components/features/BrainDetail/BrainDetailLayout";
import { InvitationsSection } from "@/components/features/BrainDetail/sections/InvitationsSection";
import type {
  BrainDetailState,
  InvitationsSectionProps,
} from "@/components/features/BrainDetail/types";
import type { ViewerState } from "@/entities/user/entity";

type BrainInvitationsPresenterProps = {
  shell: ViewerState;
  detail: BrainDetailState;
  pageCount?: number;
  invitations: InvitationsSectionProps;
};

export function BrainInvitationsPresenter({
  shell,
  detail,
  pageCount,
  invitations,
}: BrainInvitationsPresenterProps) {
  return (
    <BrainDetailLayout
      shell={shell}
      detail={detail}
      pageCount={pageCount}
      inviteCount={invitations.invitations?.length}
    >
      <InvitationsSection {...invitations} />
    </BrainDetailLayout>
  );
}
