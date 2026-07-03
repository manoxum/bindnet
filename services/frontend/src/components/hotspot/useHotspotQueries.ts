import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { HotspotBlockedDevice } from "@/components/hotspot/HotspotBlocklistCard";
import type { HotspotClient } from "@/components/hotspot/HotspotClientsCard";

export interface HotspotStatus {
  running: boolean;
  status: string;
  channel?: string;
  band?: string;
}
export interface NetworkInterface {
  name: string;
  type: "wifi" | "other";
  state: string;
  speedMbps?: number;
}

export function useHotspotQueries() {
  const status = useQuery<HotspotStatus>({
    queryKey: ["hotspot", "status"],
    queryFn: () => api.get<HotspotStatus>("/hotspot/status"),
    refetchInterval: 5000,
  });
  const config = useQuery<Record<string, string>>({
    queryKey: ["hotspot", "config"],
    queryFn: () => api.get<Record<string, string>>("/hotspot/config"),
  });
  const interfaces = useQuery<NetworkInterface[]>({
    queryKey: ["hotspot", "interfaces"],
    queryFn: () => api.get<NetworkInterface[]>("/hotspot/interfaces"),
  });
  const clients = useQuery<HotspotClient[]>({
    queryKey: ["hotspot", "clients"],
    queryFn: () => api.get<HotspotClient[]>("/hotspot/clients"),
    refetchInterval: 5000,
    enabled: !!status.data?.running,
  });
  const blocklist = useQuery<HotspotBlockedDevice[]>({
    queryKey: ["hotspot", "blocklist"],
    queryFn: () => api.get<HotspotBlockedDevice[]>("/hotspot/blocklist"),
  });

  return { status, config, interfaces, clients, blocklist };
}
