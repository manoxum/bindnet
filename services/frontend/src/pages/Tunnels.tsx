import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { api } from "@/lib/api";
import { usePageHeader } from "@/hooks/usePageHeader";
import { TunnelAccountProvider } from "@/components/tunnels/useTunnelAccount";
import { TunnelConfigCard } from "@/components/tunnels/TunnelConfigCard";
import { TunnelAccounts } from "@/components/tunnels/TunnelAccounts";
import { TunnelWorkspace } from "@/components/tunnels/TunnelWorkspace";
import type { TunnelConfig } from "@/components/tunnels/types";

export function TunnelsPage() {
  usePageHeader({
    title: "Túneis Pangolin",
    description: "Pangolin Cloud → Bindnet → nginx → aplicação local.",
  });
  const accounts = useQuery({
    queryKey: ["tunnels", "accounts"],
    queryFn: () => api.get<TunnelConfig[]>("/tunnels/accounts"),
  });
  const [params, setParams] = useSearchParams();
  const requested = params.get("account") ?? "";
  const selected = accounts.data?.find((value) => value.id === requested)?.id ?? accounts.data?.[0]?.id ?? "";
  function select(id: string) {
    setParams((current) => {
      const next = new URLSearchParams(current);
      next.set("account", id);
      return next;
    });
  }
  if (accounts.isLoading) return <p className="text-sm text-muted-foreground">Carregando contas Pangolin…</p>;
  if (accounts.isError)
    return (
      <p role="alert" className="text-sm text-destructive">
        {accounts.error.message}
      </p>
    );
  if (!accounts.data?.length) return <TunnelConfigCard onCreated={(value) => select(value.id)} />;
  return (
    <div className="space-y-6">
      <TunnelAccounts accounts={accounts.data} selected={selected} select={select} />
      <TunnelAccountProvider key={selected} accountId={selected}>
        <TunnelWorkspace />
      </TunnelAccountProvider>
    </div>
  );
}
