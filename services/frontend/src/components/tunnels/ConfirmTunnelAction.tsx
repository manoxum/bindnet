import { ConfirmDialog } from "@/components/ui/confirm-dialog";

export interface TunnelConfirmation {
  title: string;
  description: string;
  action: () => void;
}
export function ConfirmTunnelAction({
  value,
  close,
}: {
  value: TunnelConfirmation | null;
  close: () => void;
}) {
  return (
    <ConfirmDialog
      open={value !== null}
      onOpenChange={(open) => {
        if (!open) close();
      }}
      title={value?.title ?? "Confirmar ação"}
      description={value?.description}
      variant="destructive"
      onConfirm={() => {
        value?.action();
        close();
      }}
    />
  );
}
