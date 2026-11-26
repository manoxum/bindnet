import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { api } from "@/lib/api";
import { usePageHeader } from "@/hooks/usePageHeader";
import { Button } from "@/components/ui/button";
import {
  TokenList,
  type IntegrationToken,
} from "@/components/integrations/TokenList";
import { TokenIssueDialog } from "@/components/integrations/TokenIssueDialog";
export function IntegrationsPage() {
  usePageHeader({
    title: "API Token",
    description: "Gira o acesso de aplicações externas à API do Bindnet.",
  });
  const qc = useQueryClient();
  const query = useQuery({
    queryKey: ["integration-tokens"],
    queryFn: () => api.get<IntegrationToken[]>("/integrations/tokens"),
  });
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const refresh = async () => {
    await qc.invalidateQueries({ queryKey: ["integration-tokens"] });
  };
  async function revoke(item: IntegrationToken) {
    if (
      !window.confirm(
        `Revogar o token ${item.name}? As aplicações que usam este token deixam de ter acesso.`,
      )
    )
      return;
    setBusy(true);
    setError("");
    try {
      await api.del(`/integrations/tokens/${item.id}`);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Não foi possível revogar.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-4 rounded-xl border p-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold">API Token</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Emita tokens independentes, acompanhe o uso e revogue acessos
            individualmente.
          </p>
        </div>
        <Button onClick={() => setOpen(true)}>
          <Plus className="mr-2" size={16} />
          Emitir token
        </Button>
      </div>
      {(error || query.isError) && (
        <p role="alert" className="text-sm text-destructive">
          {error || "Não foi possível carregar os tokens."}
        </p>
      )}
      {query.isLoading ? (
        <p className="text-sm text-muted-foreground">A carregar tokens...</p>
      ) : (
        <TokenList
          items={query.data ?? []}
          busy={busy}
          onRevoke={(item) => void revoke(item)}
        />
      )}
      <p className="text-xs text-muted-foreground">
        Os tokens autorizam a API de certificados. Não autorizam operações de
        hotspot ou rede.
      </p>
      {open && (
        <TokenIssueDialog onClose={() => setOpen(false)} onIssued={refresh} />
      )}
    </div>
  );
}
