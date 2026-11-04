import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { TunnelConfigCard } from "./TunnelConfigCard";
import { ConfirmTunnelAction } from "./ConfirmTunnelAction";
import { useTunnelMutation } from "./useTunnels";
import type { TunnelConfig } from "./types";

export function TunnelAccounts({
  accounts,
  selected,
  select,
}: {
  accounts: TunnelConfig[];
  selected: string;
  select: (id: string) => void;
}) {
  const [adding, setAdding] = useState(false);
  const [removing, setRemoving] = useState(false);
  const account = accounts.find((value) => value.id === selected);
  const remove = useTunnelMutation(
    (id: string) => api.del(`/tunnels/accounts/${id}`),
    "Conta removida do Bindnet.",
  );
  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex-1 space-y-2">
          <Label htmlFor="tunnel-account">Conta Pangolin Cloud</Label>
          <SelectNative id="tunnel-account" value={selected} onChange={(event) => select(event.target.value)}>
            {accounts.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name} · {item.orgId}
              </option>
            ))}
          </SelectNative>
        </div>
        <Button variant="outline" onClick={() => setAdding(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Adicionar conta
        </Button>
        <Button
          variant="outline"
          disabled={!account || remove.isPending}
          aria-label="Remover conta selecionada"
          onClick={() => setRemoving(true)}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        Sites, recursos, domínios e conectores pertencem à conta selecionada. Os limites de cada plano são
        validados pelo Pangolin.
      </p>
      <Dialog open={adding} onOpenChange={setAdding}>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>Nova conta Pangolin Cloud</DialogTitle>
            <DialogDescription>Use a chave da organização que pretende gerir nesta conta.</DialogDescription>
          </DialogHeader>
          <TunnelConfigCard
            onCreated={(value) => {
              select(value.id);
              setAdding(false);
            }}
          />
        </DialogContent>
      </Dialog>
      <ConfirmTunnelAction
        value={
          removing && account
            ? {
                title: `Remover ${account.name}?`,
                description:
                  "Remova primeiro os sites e recursos geridos desta conta. A conta no Pangolin Cloud continuará a existir.",
                action: () => remove.mutate(account.id),
              }
            : null
        }
        close={() => setRemoving(false)}
      />
    </div>
  );
}
