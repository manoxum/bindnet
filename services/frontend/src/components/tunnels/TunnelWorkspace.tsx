import { Cable, Globe, RefreshCw, Settings } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";

import { useUrlTab } from "@/hooks/useUrlTab";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { TunnelConfigCard } from "@/components/tunnels/TunnelConfigCard";
import { TunnelSites } from "@/components/tunnels/TunnelSites";
import { TunnelPublicationDialog } from "./TunnelPublicationDialog";
import { TunnelResourceList } from "@/components/tunnels/TunnelResourceList";
import { useTunnels } from "@/components/tunnels/useTunnels";

export function TunnelWorkspace() {
  const { config, sites, resources, domains } = useTunnels();
  const [tab, setTab] = useUrlTab("sites");
  const client = useQueryClient();
  const configured = config.data?.configured === true;
  const errors = [config.error, sites.error, resources.error, domains.error].filter(Boolean);

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 rounded-lg border border-primary/20 bg-primary/5 p-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <p className="text-xs font-semibold uppercase tracking-widest text-primary">
            Pangolin → Bindnet → nginx → aplicação
          </p>
          <p className="text-sm text-muted-foreground">
            Informe o nome público e o domínio local. O Bindnet cria o recurso e inicia o conector
            automaticamente.
          </p>
        </div>
        <Button
          variant="outline"
          className="shrink-0"
          disabled={sites.isFetching || resources.isFetching}
          onClick={() => client.invalidateQueries({ queryKey: ["tunnels"] })}
        >
          <RefreshCw className={`mr-2 h-4 w-4 ${sites.isFetching ? "animate-spin" : ""}`} />
          Atualizar
        </Button>
      </div>
      {errors.map((error, index) => (
        <p
          key={index}
          role="alert"
          className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive"
        >
          {error?.message}
        </p>
      ))}
      {config.isLoading ? (
        <p className="text-sm text-muted-foreground">Carregando integração…</p>
      ) : !configured ? (
        <TunnelConfigCard config={config.data} />
      ) : (
        <Tabs value={tab} onValueChange={setTab} className="space-y-4">
          <TabsList className="grid h-auto w-full grid-cols-3 sm:inline-grid sm:w-auto">
            <TabsTrigger value="services" className="gap-2">
              <Globe className="h-4 w-4" />
              Serviços
            </TabsTrigger>
            <TabsTrigger value="sites" className="gap-2">
              <Cable className="h-4 w-4" />
              Sites
            </TabsTrigger>
            <TabsTrigger value="config" className="gap-2">
              <Settings className="h-4 w-4" />
              Integração
            </TabsTrigger>
          </TabsList>
          <TabsContent value="services" className="space-y-4">
            {sites.isLoading || domains.isLoading ? (
              <p className="text-sm text-muted-foreground">Consultando sites e domínios…</p>
            ) : (
              !sites.isError &&
              !domains.isError && (
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <h2 className="text-lg font-semibold">Recursos publicados</h2>
                  <TunnelPublicationDialog sites={sites.data ?? []} domains={domains.data ?? []} />
                </div>
              )
            )}
            {resources.isLoading ? (
              <p className="text-sm text-muted-foreground">Carregando serviços…</p>
            ) : (
              !resources.isError && <TunnelResourceList resources={resources.data ?? []} />
            )}
          </TabsContent>
          <TabsContent value="sites">
            {sites.isLoading ? (
              <p className="text-sm text-muted-foreground">Carregando sites…</p>
            ) : (
              !sites.isError && <TunnelSites sites={sites.data ?? []} domains={domains.data ?? []} />
            )}
          </TabsContent>
          <TabsContent value="config">
            <TunnelConfigCard config={config.data} />
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}
