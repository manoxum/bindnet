import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { PortalMessage } from "@/components/portal/portal-types";

// Avisos direcionados ao dispositivo (ou broadcast) que ele ainda nao
// marcou como lidos. Rota publica: o backend identifica o MAC pelo IP de
// origem, nunca por um MAC informado aqui (ver usePortalMe).
export function usePortalMessages() {
  return useQuery<PortalMessage[]>({
    queryKey: ["portal", "messages"],
    queryFn: () => api.get<PortalMessage[]>("/hotspot/portal/messages"),
    retry: false,
    refetchInterval: 15000,
  });
}

export function useMarkMessageRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.post<void>(`/hotspot/portal/messages/${id}/read`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["portal", "messages"] }),
  });
}
