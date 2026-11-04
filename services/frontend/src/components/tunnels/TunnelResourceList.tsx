import { lazy, Suspense, useState } from "react";
import { ExternalLink, Pencil, Power, Trash2, Settings2 } from "lucide-react";
import { useTunnelApi } from "./useTunnelAccount";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ConfirmTunnelAction, type TunnelConfirmation } from "./ConfirmTunnelAction";
import { TunnelTargetDialog } from "./TunnelTargetDialog";
import { useTunnelMutation } from "./useTunnels";
import type { TunnelResource } from "./types";

const TunnelResourceSettings = lazy(() => import("./TunnelResourceSettings"));

export function TunnelResourceList({ resources }: { resources: TunnelResource[] }) {
  const api = useTunnelApi();
  const [confirmation, setConfirmation] = useState<TunnelConfirmation | null>(null);
  const [configuring, setConfiguring] = useState<TunnelResource | null>(null);
  const [editing, setEditing] = useState<TunnelResource | null>(null);
  const toggle = useTunnelMutation(
    (resource: TunnelResource) =>
      api.patch(`/tunnels/resources/${resource.resourceId}`, { enabled: !resource.enabled }),
    "Estado do recurso atualizado.",
  );
  const remove = useTunnelMutation((id: number) => api.del(`/tunnels/resources/${id}`), "Recurso removido.");
  const busy = toggle.isPending || remove.isPending;
  return (
    <Card>
      <CardHeader>
        <CardTitle>Serviços no Pangolin</CardTitle>
        <CardDescription>
          Os recursos criados fora deste Bindnet aparecem apenas para consulta. Políticas de acesso são
          geridas no Pangolin.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {resources.length === 0 && (
          <p className="py-6 text-center text-sm text-muted-foreground">
            Nenhum serviço publicado nesta organização.
          </p>
        )}
        {resources.map((resource) => (
          <div
            key={resource.resourceId}
            className="flex flex-col gap-4 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div className="min-w-0 space-y-2">
              <div className="flex flex-wrap items-center gap-2">
                <span className="break-all font-medium">{resource.name}</span>
                <Badge variant={resource.enabled ? "secondary" : "outline"}>
                  {resource.enabled ? "Ativo" : "Desativado"}
                </Badge>
                <Badge variant="outline">{resource.mode.toUpperCase() || "PENDENTE"}</Badge>
              </div>
              {resource.fullDomain && /^[a-z0-9.-]+$/i.test(resource.fullDomain) ? (
                <a
                  className="inline-flex max-w-full items-center gap-1 text-sm text-primary underline underline-offset-4"
                  href={`https://${resource.fullDomain}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  <span className="break-all">{resource.fullDomain}</span>
                  <ExternalLink className="h-3 w-3 shrink-0" />
                </a>
              ) : (
                <p className="text-sm text-muted-foreground">
                  {resource.proxyPort
                    ? `Porta pública ${resource.proxyPort}`
                    : `Recurso #${resource.resourceId}`}
                </p>
              )}
              {resource.localDomain && (
                <p className="break-all text-sm text-muted-foreground">
                  nginx → {resource.localDomain} → aplicação
                </p>
              )}
              <p className="break-all font-mono text-xs text-muted-foreground">
                {resource.targets?.map((target) => `${target.ip}:${target.port}`).join(" · ") ||
                  "Sem destinos disponíveis"}
              </p>
              <p className="text-xs text-muted-foreground">
                {resource.mode === "http"
                  ? resource.sso
                    ? "Protegido por autenticação Pangolin"
                    : "SSO desativado no Pangolin"
                  : "Sem autenticação Pangolin para TCP/UDP"}{" "}
                · {resource.managed ? "Gerido pelo Bindnet" : "Somente leitura"}
              </p>
            </div>
            {resource.managed && (
              <div className="flex shrink-0 flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  aria-label={`Configurar ${resource.name}`}
                  onClick={() => setConfiguring(resource)}
                >
                  <Settings2 className="h-4 w-4 sm:mr-2" />
                  <span className="hidden sm:inline">Configurações</span>
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={busy}
                  aria-label={`Editar destino de ${resource.name}`}
                  onClick={() => setEditing(resource)}
                >
                  <Pencil className="h-4 w-4 sm:mr-2" />
                  <span className="hidden sm:inline">Destino</span>
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={busy}
                  aria-label={`${resource.enabled ? "Desativar" : "Ativar"} ${resource.name}`}
                  onClick={() =>
                    setConfirmation({
                      title: `${resource.enabled ? "Desativar" : "Ativar"} ${resource.name}?`,
                      description: resource.enabled
                        ? "O serviço ficará inacessível pelo túnel."
                        : resource.mode === "http"
                          ? "O serviço local ficará acessível pelo domínio público, com as políticas de acesso configuradas no Pangolin."
                          : "O serviço local ficará acessível publicamente pela porta remota, sem autenticação Pangolin.",
                      action: () => toggle.mutate(resource),
                    })
                  }
                >
                  <Power className="h-4 w-4 sm:mr-2" />
                  <span className="hidden sm:inline">{resource.enabled ? "Desativar" : "Ativar"}</span>
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={busy}
                  aria-label={`Remover recurso ${resource.name}`}
                  onClick={() =>
                    setConfirmation({
                      title: `Remover ${resource.name}?`,
                      description:
                        "O recurso e seus destinos serão removidos no Pangolin. A aplicação local continuará em execução.",
                      action: () => remove.mutate(resource.resourceId),
                    })
                  }
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            )}
          </div>
        ))}
        <ConfirmTunnelAction value={confirmation} close={() => setConfirmation(null)} />
        {configuring && (
          <Suspense fallback={<p className="text-sm text-muted-foreground">Carregando configurações…</p>}>
            <TunnelResourceSettings resource={configuring} close={() => setConfiguring(null)} />
          </Suspense>
        )}
        <TunnelTargetDialog resource={editing} close={() => setEditing(null)} />
      </CardContent>
    </Card>
  );
}
