import { TabsContent } from "@/components/ui/tabs";
import { HotspotBlocklistCard } from "@/components/hotspot/HotspotBlocklistCard";
import { HotspotClientsCard } from "@/components/hotspot/HotspotClientsCard";
import { HotspotKnownDevicesCard } from "@/components/hotspot/HotspotKnownDevicesCard";
import { HotspotIsolationCard } from "@/components/hotspot/HotspotIsolationCard";
import { HotspotContentCard } from "@/components/hotspot/HotspotContentCard";
import { HotspotProfilesCard } from "@/components/hotspot/HotspotProfilesCard";
import { HotspotVouchersCard } from "@/components/hotspot/HotspotVouchersCard";
import { HotspotMessagesCard } from "@/components/hotspot/messages/HotspotMessagesCard";
import { useHotspotQueries } from "@/components/hotspot/useHotspotQueries";
import { useHotspotMutations } from "@/components/hotspot/useHotspotMutations";
import { LogsPanel } from "@/components/LogsPanel";

interface HotspotTabsContentProps {
  queries: ReturnType<typeof useHotspotQueries>;
  mutations: ReturnType<typeof useHotspotMutations>;
  blockedMacs: Set<string>;
}

// Conteudo das abas da tela de hotspot - extraido de pages/Hotspot.tsx
// para manter aquele arquivo dentro do limite de ~200 linhas deste repo
// (mesmo motivo/precedente de HotspotTabsList.tsx). Continua dentro de
// <Tabs> no componente pai: TabsContent (Radix) usa contexto React, que
// atravessa normalmente essa borda de componente.
export function HotspotTabsContent({ queries, mutations, blockedMacs }: HotspotTabsContentProps) {
  const { status, clients, blocklist, knownDevices } = queries;
  const { block, unblock, clearLogs } = mutations;

  return (
    <>
      <TabsContent value="connected" className="mt-0">
        <HotspotClientsCard
          clients={clients.data ?? []}
          running={!!status.data?.running}
          blockPendingMac={block.isPending ? block.variables.mac : undefined}
          unblockPendingMac={unblock.isPending ? unblock.variables : undefined}
          onBlock={(mac, mode) => block.mutate({ mac, mode })}
          onUnblock={(mac) => unblock.mutate(mac)}
        />
      </TabsContent>

      <TabsContent value="blocked" className="mt-0">
        <HotspotBlocklistCard
          devices={blocklist.data ?? []}
          unblockPendingMac={unblock.isPending ? unblock.variables : undefined}
          onUnblock={(mac) => unblock.mutate(mac)}
        />
      </TabsContent>

      <TabsContent value="known" className="mt-0">
        <HotspotKnownDevicesCard
          devices={knownDevices.data ?? []}
          blockedMacs={blockedMacs}
          blockPendingMac={block.isPending ? block.variables.mac : undefined}
          unblockPendingMac={unblock.isPending ? unblock.variables : undefined}
          onBlock={(mac, mode) => block.mutate({ mac, mode })}
          onUnblock={(mac) => unblock.mutate(mac)}
        />
      </TabsContent>

      <TabsContent value="profiles" className="mt-0">
        <HotspotProfilesCard />
      </TabsContent>

      <TabsContent value="isolation" className="mt-0">
        <HotspotIsolationCard knownDevices={knownDevices.data ?? []} />
      </TabsContent>

      <TabsContent value="content" className="mt-0">
        <HotspotContentCard />
      </TabsContent>

      <TabsContent value="vouchers" className="mt-0">
        <HotspotVouchersCard />
      </TabsContent>

      <TabsContent value="messages" className="mt-0">
        <HotspotMessagesCard />
      </TabsContent>

      <TabsContent value="logs" className="mt-0">
        <LogsPanel title="Logs do hotspot" path="/hotspot/logs" onClear={() => clearLogs.mutateAsync()} />
      </TabsContent>
    </>
  );
}
