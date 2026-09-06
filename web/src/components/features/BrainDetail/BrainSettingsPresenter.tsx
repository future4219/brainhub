import { BrainDetailLayout } from "@/components/features/BrainDetail/BrainDetailLayout";
import type { BrainDetailState } from "@/components/features/BrainDetail/types";
import { Button } from "@/components/ui/Button";
import { Panel } from "@/components/ui/Panel";
import type { ViewerState } from "@/entities/user/entity";

export function BrainSettingsPresenter(props: {
  shell: ViewerState;
  detail: BrainDetailState;
  canReissueWriter: boolean;
  writerReissuing: boolean;
  writerStatus: string;
  onReissueWriter: () => void;
}) {
  return (
    <BrainDetailLayout shell={props.shell} detail={props.detail}>
      <Panel className="mt-6 p-4">
        <h2 className="text-body font-semibold">脳への接続を復旧</h2>
        <p className="mt-2 text-ui text-text-secondary">
          この脳のページを表示・保存できなくなった場合に、オーナーが脳専用の接続を再発行できます。
        </p>
        {props.canReissueWriter ? (
          <Button
            className="mt-4"
            type="button"
            disabled={props.writerReissuing}
            onClick={props.onReissueWriter}
          >
            {props.writerReissuing ? "復旧中…" : "脳専用の接続を再発行"}
          </Button>
        ) : (
          <p className="mt-4 text-ui text-text-muted">
            設定の変更にはオーナー権限が必要です。
          </p>
        )}
        {props.writerStatus && (
          <p className="mt-3 text-ui" role="status">
            {props.writerStatus}
          </p>
        )}
      </Panel>
    </BrainDetailLayout>
  );
}
