import { useQuery } from "@tanstack/react-query";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ResourceSettingsForm } from "./ResourceSettingsForm";
import { ResourceAuthMethods } from "./ResourceAuthMethods";
import { ResourceMembership } from "./ResourceMembership";
import { ResourceMtls } from "./ResourceMtls";
import { ResourceRules } from "./ResourceRules";
import { ResourcePublicAddress } from "./ResourcePublicAddress";
import { useTunnelApi, useTunnelAccount } from "./useTunnelAccount";
import {
  authFields,
  generalFields,
  maintenanceFields,
  proxyFields,
  type ResourceSettings,
} from "./resource-settings-types";
import type { TunnelResource } from "./types";

export default function TunnelResourceSettings({
  resource,
  close,
}: {
  resource: TunnelResource;
  close: () => void;
}) {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const settings = useQuery({
    queryKey: ["tunnels", account, "settings", resource.resourceId],
    queryFn: () => api.get<ResourceSettings>(`/tunnels/resources/${resource.resourceId}/settings`),
    retry: false,
  });
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) close();
      }}
    >
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Configurações de {resource.name}</DialogTitle>
          <DialogDescription>
            Alterações são aplicadas à conta Pangolin selecionada. O domínio local continua a definir o alvo
            nginx.
          </DialogDescription>
        </DialogHeader>
        {settings.isLoading ? (
          <p className="text-sm text-muted-foreground">Carregando configurações do Pangolin…</p>
        ) : settings.isError ? (
          <p role="alert" className="text-sm text-destructive">
            {settings.error.message}
          </p>
        ) : (
          settings.data && (
            <Tabs defaultValue="general" className="space-y-5">
              <TabsList className="grid h-auto w-full grid-cols-2 sm:inline-grid sm:w-auto sm:grid-cols-5">
                <TabsTrigger value="general">Gerais</TabsTrigger>
                <TabsTrigger value="http">HTTP(S)</TabsTrigger>
                <TabsTrigger value="auth">Autenticação</TabsTrigger>
                <TabsTrigger value="rules">Regras de acesso</TabsTrigger>
                <TabsTrigger value="maintenance" className="col-span-2 sm:col-span-1">
                  Manutenção
                </TabsTrigger>
              </TabsList>
              <TabsContent value="general" className="space-y-5">
                <p className="break-all text-sm text-primary">
                  {String(settings.data.fullDomain ?? resource.fullDomain)}
                </p>
                <ResourceSettingsForm
                  resourceId={resource.resourceId}
                  settings={settings.data}
                  fields={generalFields}
                />
                <ResourcePublicAddress resourceId={resource.resourceId} />
              </TabsContent>
              <TabsContent value="http" className="space-y-5">
                <div className="rounded border p-3 text-sm">
                  <p className="break-all">
                    Host nginx:{" "}
                    {String(settings.data.setHostHeader ?? resource.localDomain ?? "não configurado")}
                  </p>
                  <p className="break-all">
                    Nome TLS / SNI: {String(settings.data.tlsServerName ?? "não utilizado")}
                  </p>
                  <p className="mt-2 text-xs text-muted-foreground">
                    Use a ação “Destino” na lista para alterar o DNS local e atualizar alvo, Host e SNI
                    juntos.
                  </p>
                </div>
                <ResourceSettingsForm
                  resourceId={resource.resourceId}
                  settings={settings.data}
                  fields={proxyFields}
                />
              </TabsContent>
              <TabsContent value="auth" className="space-y-5">
                <ResourceSettingsForm
                  resourceId={resource.resourceId}
                  settings={settings.data}
                  fields={authFields}
                />
                <ResourceMembership resourceId={resource.resourceId} kind="roles" />
                <ResourceMembership resourceId={resource.resourceId} kind="users" />
                <ResourceAuthMethods resourceId={resource.resourceId} />
                <ResourceMtls resourceId={resource.resourceId} />
              </TabsContent>
              <TabsContent value="rules">
                <ResourceRules resourceId={resource.resourceId} settings={settings.data} />
              </TabsContent>
              <TabsContent value="maintenance">
                <ResourceSettingsForm
                  resourceId={resource.resourceId}
                  settings={settings.data}
                  fields={maintenanceFields}
                />
              </TabsContent>
            </Tabs>
          )
        )}
      </DialogContent>
    </Dialog>
  );
}
