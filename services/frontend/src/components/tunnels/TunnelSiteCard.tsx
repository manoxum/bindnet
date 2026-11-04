import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Cable, Play, Square, Trash2 } from "lucide-react";
import { useTunnelApi, useTunnelAccount } from "./useTunnelAccount";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ConfirmTunnelAction, type TunnelConfirmation } from "./ConfirmTunnelAction";
import { useTunnelMutation } from "./useTunnels";
import type { ConnectorStatus, TunnelSite } from "./types";

export function TunnelSiteCard({ site }: { site: TunnelSite }) {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const [confirmation, setConfirmation] = useState<TunnelConfirmation | null>(null);
  const status = useQuery({
    queryKey: ["tunnels", account, "connector", site.siteId],
    queryFn: () => api.get<ConnectorStatus>(`/tunnels/sites/${site.siteId}/status`),
    enabled: site.managed,
    refetchInterval: 15000,
    retry: false,
  });
  const action = useTunnelMutation(
    (value: "start" | "stop") => api.post(`/tunnels/sites/${site.siteId}/${value}`),
    "Comando enviado ao conector.",
  );
  const remove = useTunnelMutation(
    () => api.del(`/tunnels/sites/${site.siteId}`),
    "Site e conector removidos.",
  );
  const busy = action.isPending || remove.isPending;
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-wrap items-center justify-between gap-2 text-base">
          <span className="flex min-w-0 items-center gap-2">
            <Cable className="h-4 w-4 shrink-0" />
            <span className="break-all">{site.name}</span>
          </span>
          <Badge variant={site.online ? "secondary" : "outline"}>
            {site.online ? "Online no Pangolin" : "Offline no Pangolin"}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-xs text-muted-foreground">
          Site #{site.siteId} · {site.managed ? "Gerido por este Bindnet" : "Site externo · somente leitura"}
        </p>
        {site.managed && (
          <>
            <p className="text-sm text-muted-foreground">
              Conector local:{" "}
              {status.isLoading
                ? "consultando…"
                : status.isError
                  ? "indisponível no worker"
                  : status.data?.running
                    ? `em execução${status.data.health === "healthy" ? " · túnel conectado" : " · aguardando conectividade"}`
                    : "parado ou ainda não instalado"}
            </p>
            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                disabled={busy || status.isLoading || status.data?.running}
                onClick={() => action.mutate("start")}
              >
                <Play className="mr-2 h-4 w-4" />
                {action.isPending ? "Aguarde…" : "Iniciar conector"}
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={busy || !status.data?.running}
                onClick={() =>
                  setConfirmation({
                    title: "Parar o conector?",
                    description: "Os serviços publicados por este site ficarão inacessíveis pelo túnel.",
                    action: () => action.mutate("stop"),
                  })
                }
              >
                <Square className="mr-2 h-4 w-4" />
                Parar
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={busy}
                aria-label={`Remover site ${site.name}`}
                onClick={() =>
                  setConfirmation({
                    title: `Remover ${site.name}?`,
                    description:
                      "Remove o site remoto e o conector local. Remova antes os recursos geridos deste site. Recursos criados fora do Bindnet podem perder seus destinos.",
                    action: () => remove.mutate(),
                  })
                }
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </>
        )}
        <ConfirmTunnelAction value={confirmation} close={() => setConfirmation(null)} />
      </CardContent>
    </Card>
  );
}
