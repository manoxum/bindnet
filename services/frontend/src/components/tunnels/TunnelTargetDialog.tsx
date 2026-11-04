import { useState } from "react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { LocalDomainFields } from "./LocalDomainFields";
import { useTunnelMutation } from "./useTunnels";
import { useTunnelApi } from "./useTunnelAccount";
import type { TunnelResource } from "./types";

function DestinationForm({ resource, close }: { resource: TunnelResource; close: () => void }) {
  const api = useTunnelApi();
  const [localDomain, setLocalDomain] = useState(resource.localDomain ?? "");
  const [method, setMethod] = useState<"http" | "https">(
    resource.targets?.[0]?.method === "http" ? "http" : "https",
  );
  const save = useTunnelMutation(
    () => api.patch(`/tunnels/resources/${resource.resourceId}/destination`, { localDomain, method }),
    "Domínio local e alvo nginx atualizados.",
  );
  return (
    <form
      className="space-y-4"
      onSubmit={(event) => {
        event.preventDefault();
        save.mutate(undefined, { onSuccess: close });
      }}
    >
      <LocalDomainFields
        localDomain={localDomain}
        method={method}
        onDomainChange={setLocalDomain}
        onMethodChange={setMethod}
      />
      <p className="text-xs text-muted-foreground">
        O Bindnet atualiza Host, SNI e alvo no Pangolin. O recurso é pausado durante a alteração e retorna ao
        estado anterior quando concluída.
      </p>
      <Button disabled={save.isPending || !localDomain.trim()}>
        {save.isPending ? "Salvando…" : "Salvar domínio local"}
      </Button>
    </form>
  );
}
export function TunnelTargetDialog({
  resource,
  close,
}: {
  resource: TunnelResource | null;
  close: () => void;
}) {
  return (
    <Dialog
      open={resource !== null}
      onOpenChange={(open) => {
        if (!open) close();
      }}
    >
      <DialogContent className="max-h-[85dvh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Domínio local de {resource?.name}</DialogTitle>
          <DialogDescription>Pangolin → Bindnet → nginx → aplicação.</DialogDescription>
        </DialogHeader>
        {resource && <DestinationForm key={resource.resourceId} resource={resource} close={close} />}
      </DialogContent>
    </Dialog>
  );
}
