import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { PublicDomainPicker } from "./PublicDomainPicker";
import { useTunnelApi, useTunnelAccount } from "./useTunnelAccount";
import { useTunnelMutation } from "./useTunnels";
import type { TunnelDomain } from "./types";
export function ResourcePublicAddress({ resourceId }: { resourceId: number }) {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const [domainId, setDomainId] = useState("");
  const [subdomain, setSubdomain] = useState("");
  const domains = useQuery({
    queryKey: ["tunnels", account, "domains"],
    queryFn: () => api.get<TunnelDomain[]>("/tunnels/domains"),
  });
  const save = useTunnelMutation(
    () => api.patch(`/tunnels/resources/${resourceId}/settings`, { domainId, subdomain: subdomain || null }),
    "Endereço público atualizado.",
  );
  return (
    <details className="rounded-lg border p-4">
      <summary className="cursor-pointer text-sm font-medium">Alterar endereço público</summary>
      <div className="mt-4 space-y-4">
        {domains.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {domains.error.message}
          </p>
        ) : (
          <>
            <PublicDomainPicker
              domains={domains.data ?? []}
              domainId={domainId}
              subdomain={subdomain}
              onDomainChange={setDomainId}
              onSubdomainChange={setSubdomain}
            />
            <p className="text-xs text-muted-foreground">
              O domínio anterior deixará de encaminhar para este recurso.
            </p>
            <Button variant="outline" disabled={save.isPending || !domainId} onClick={() => save.mutate()}>
              Alterar endereço público
            </Button>
          </>
        )}
      </div>
    </details>
  );
}
