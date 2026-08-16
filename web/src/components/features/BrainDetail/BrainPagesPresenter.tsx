import { BrainDetailLayout } from "@/components/features/BrainDetail/BrainDetailLayout";
import { PagesSection } from "@/components/features/BrainDetail/sections/PagesSection";
import type {
  BrainDetailState,
  PagesSectionProps,
} from "@/components/features/BrainDetail/types";
import type { ViewerState } from "@/entities/user/entity";

type BrainPagesPresenterProps = {
  shell: ViewerState;
  detail: BrainDetailState;
  page: PagesSectionProps;
};

export function BrainPagesPresenter({
  shell,
  detail,
  page,
}: BrainPagesPresenterProps) {
  return (
    <BrainDetailLayout
      shell={shell}
      detail={detail}
      pageCount={page.pages?.length}
      search={{
        value: page.query,
        placeholder: "ページを検索",
        onChange: page.onQuery,
      }}
    >
      <PagesSection {...page} />
    </BrainDetailLayout>
  );
}
