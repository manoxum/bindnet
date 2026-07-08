import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { PortalCreditHistoryEntry, PortalMeResponse } from "@/components/portal/portal-types";

// Rotas publicas (sem sessao) - o backend identifica o dispositivo
// pelo IP de origem, nunca por um MAC informado pelo cliente (ver
// services/backend/hotspot_portal.go).
export function usePortalMe() {
  return useQuery<PortalMeResponse>({
    queryKey: ["portal", "me"],
    queryFn: () => api.get<PortalMeResponse>("/hotspot/portal/me"),
    retry: false,
    refetchInterval: 15000,
  });
}

export function usePortalCreditHistory(enabled: boolean) {
  return useQuery<PortalCreditHistoryEntry[]>({
    queryKey: ["portal", "credit", "history"],
    queryFn: () => api.get<PortalCreditHistoryEntry[]>("/hotspot/portal/credit/history"),
    enabled,
  });
}
